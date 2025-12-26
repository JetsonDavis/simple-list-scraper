package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/playwright-community/playwright-go"
)

var (
    db            *sql.DB
    workerRunning atomic.Bool
    ollamaCmd     *exec.Cmd
    wsClients     = make(map[*websocket.Conn]bool)
    wsClientsMux  sync.Mutex
    wsUpgrader    = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true // Allow all origins for development
        },
    }
)

type Item struct {
    ID   int64  `json:"id"`
    Text string `json:"text"`
}

type URL struct {
    ID          int64  `json:"id"`
    URL         string `json:"url"`
    DisplayName string `json:"display_name,omitempty"`
    Config      string `json:"config,omitempty"` // JSON string for scraper config
}

type SearchResult struct {
    Title      string
    URL        string
    MagnetLink string
}

type Entity struct {
    Text       string  `json:"text"`
    Type       string  `json:"type"`
    Start      int     `json:"start"`
    End        int     `json:"end"`
    Confidence float64 `json:"confidence"`
}

type EntityExtractionResponse struct {
    Entities []Entity `json:"entities"`
}

type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
    Format string `json:"format,omitempty"`
}

type OllamaResponse struct {
    Response string `json:"response"`
}

type SiteScraper interface {
    Name() string
    Search(ctx context.Context, pw *playwright.Playwright, query string) ([]SearchResult, error)
}

func main() {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL not set (e.g. postgres://user:password@localhost:5432/dbname?sslmode=disable)")
    }

    intervalHours := getenvInt("CHECK_INTERVAL_HOURS", 6)
    if intervalHours <= 0 {
        intervalHours = 6
    }
    interval := time.Duration(intervalHours) * time.Hour

    runOnStart := strings.ToLower(os.Getenv("RUN_WORKER_ON_START"))
    if runOnStart == "" {
        runOnStart = "false"
    }

    var err error
    db, err = sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatal(err)
    }
    if err := initDB(db); err != nil {
        log.Fatal(err)
    }

    // Check Ollama availability if entity matching is enabled
    useEntityMatching := strings.ToLower(os.Getenv("USE_ENTITY_MATCHING")) == "true"
    if useEntityMatching {
        log.Println("Entity matching enabled, starting Ollama if needed...")

        // Try to start Ollama if it's not running
        if err := startOllama(); err != nil {
            log.Printf("WARNING: Failed to start Ollama: %v", err)
            log.Println("Entity extraction will be skipped. To fix:")
            log.Println("  1. Manually start Ollama: ollama serve")
            log.Println("  2. Pull the model: ollama pull " + os.Getenv("OLLAMA_MODEL"))
            log.Println("  3. Or disable entity matching: USE_ENTITY_MATCHING=false")
        } else {
            // Now check health and initialize the model
            if err := checkOllamaHealth(); err != nil {
                log.Printf("WARNING: Ollama health check failed: %v", err)
                log.Println("Entity extraction will be skipped.")
            } else {
                ollamaModel := os.Getenv("OLLAMA_MODEL")
                if ollamaModel == "" {
                    ollamaModel = "llama2"
                }
                log.Println("========================================")
                log.Printf("✓ OLLAMA MODEL %q IS RUNNING AND READY", strings.ToUpper(ollamaModel))
                log.Println("✓ Entity extraction is enabled and operational")
                log.Println("========================================")
            }
        }
    }

    go scheduler(interval, runOnStart == "true")

    mux := http.NewServeMux()
    mux.HandleFunc("/api/items", itemsHandler)
    mux.HandleFunc("/api/items/", itemHandler)
    mux.HandleFunc("/api/urls", urlsHandler)
    mux.HandleFunc("/api/urls/", urlHandler)
    mux.HandleFunc("/api/matches", matchesHandler)
    mux.HandleFunc("/api/matches/", matchHandler)
    mux.HandleFunc("/api/logs", logsHandler)
    mux.HandleFunc("/api/trigger-worker", triggerWorkerHandler)
    mux.HandleFunc("/api/worker-status", workerStatusHandler)
    mux.HandleFunc("/api/ws", wsHandler)
    mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, map[string]any{"status": "ok"})
    })

    server := &http.Server{
        Addr:              "127.0.0.1:8004",
        Handler:           loggingMiddleware(mux),
        ReadHeaderTimeout: 5 * time.Second,
    }

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    go func() {
        log.Println("API listening on 127.0.0.1:8004")
        if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Fatal(err)
        }
    }()

    <-ctx.Done()
    log.Println("Shutting down...")

    // Stop Ollama if we started it
    if ollamaCmd != nil && ollamaCmd.Process != nil {
        log.Println("Stopping Ollama server...")
        if err := ollamaCmd.Process.Kill(); err != nil {
            log.Printf("Failed to stop Ollama: %v", err)
        } else {
            log.Println("Ollama server stopped")
        }
    }

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = server.Shutdown(shutdownCtx)
}

func getenvInt(key string, def int) int {
    v := strings.TrimSpace(os.Getenv(key))
    if v == "" {
        return def
    }
    n, err := strconv.Atoi(v)
    if err != nil {
        return def
    }
    return n
}

func getenvFloat(key string, def float64) float64 {
    v := strings.TrimSpace(os.Getenv(key))
    if v == "" {
        return def
    }
    f, err := strconv.ParseFloat(v, 64)
    if err != nil {
        return def
    }
    if f < 0 {
        return 0
    }
    if f > 1 {
        return 1
    }
    return f
}

func scheduler(interval time.Duration, runOnStart bool) {
    if runOnStart {
        runWorker()
    }
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for range ticker.C {
        runWorker()
    }
}

func runWorker() {
    if !workerRunning.CompareAndSwap(false, true) {
        log.Println("Worker already running, skipping")
        return
    }
    defer func() {
        workerRunning.Store(false)
        broadcastWorkerStatus("completed", "Worker finished")
    }()

    disablePW := strings.ToLower(os.Getenv("DISABLE_PLAYWRIGHT")) == "true"
    threshold := getenvFloat("FUZZY_THRESHOLD", 0.78)

    log.Printf("Worker started (threshold=%.2f, playwright_disabled=%v)\n", threshold, disablePW)
    broadcastWorkerStatus("running", "Worker started")

    items, err := loadItems()
    if err != nil {
        log.Println("worker load items:", err)
        return
    }
    if len(items) == 0 {
        log.Println("worker: no items; done")
        return
    }

    urls, err := loadUrls()
    if err != nil {
        log.Println("worker load urls:", err)
        return
    }
    if len(urls) == 0 {
        log.Println("worker: no urls; done")
        return
    }

    scrapers := []SiteScraper{}
    for _, u := range urls {
        displayName := u.DisplayName
        if displayName == "" {
            displayName = u.URL
        }
        scrapers = append(scrapers, &GenericScraper{URL: u.URL, DisplayName: displayName, Config: u.Config})
    }

    var pw *playwright.Playwright
    if !disablePW {
        if err := playwright.Install(); err != nil {
            // In container builds we already install browsers; this is a fallback.
            log.Println("playwright.Install warning:", err)
        }
        pw, err = playwright.Run()
        if err != nil {
            log.Println("playwright.Run error:", err)
            return
        }
        defer func() {
            _ = pw.Stop()
        }()
    }

    for _, it := range items {
        matchesFound := 0
        maxMatchesPerItem := 5

        for _, s := range scrapers {
            // Check if we've already found enough matches for this item
            if matchesFound >= maxMatchesPerItem {
                log.Printf("Found %d matches for item %q, moving to next item\n", matchesFound, it.Text)
                break
            }

            results, err := s.Search(context.Background(), pw, it.Text)
            if err != nil {
                log.Printf("scraper %s error: %v\n", s.Name(), err)
                continue
            }
            log.Printf("Scraper %s returned %d results for item %q\n", s.Name(), len(results), it.Text)
            for i, r := range results {
                log.Printf("  Result %d: title=%q url=%s has_magnet=%v\n", i+1, r.Title, r.URL, r.MagnetLink != "")
                // Check if we've reached the limit during result processing
                if matchesFound >= maxMatchesPerItem {
                    break
                }

                // Check quality FIRST before any other processing
                if disqualifiedQuality(r.Title) {
                    log.Printf("DISQUALIFIED_QUALITY site=%s url=%s title=%q - skipping\n", s.Name(), r.URL, r.Title)
                    continue
                }

                // Extract year from item text
                itemYear := extractYear(it.Text)
                itemWithoutYear := removeYear(it.Text)

                // Log item year extraction
                if itemYear != "" {
                    log.Printf("Item year extracted: %q has year=%s (without year: %q)\n", it.Text, itemYear, itemWithoutYear)
                } else {
                    log.Printf("Item has no year: %q\n", it.Text)
                }

                // Log the scraped torrent title before processing
                log.Printf("Scraped from page: title=%q url=%s\n", r.Title, r.URL)

                // Pre-filter: Check if item text appears as contiguous phrase in result
                // Normalize both strings and check if item words appear together in order
                normalizedItem := normalize(removeYear(it.Text))
                normalizedTitle := normalize(r.Title)

                // Check if the item appears as a contiguous phrase (allowing for extra chars like dots, dashes)
                // Replace common separators with spaces for matching
                titleForMatching := strings.ReplaceAll(normalizedTitle, ".", " ")
                titleForMatching = strings.ReplaceAll(titleForMatching, "-", " ")
                titleForMatching = strings.ReplaceAll(titleForMatching, "_", " ")

                itemForMatching := strings.ReplaceAll(normalizedItem, ".", " ")
                itemForMatching = strings.ReplaceAll(itemForMatching, "-", " ")
                itemForMatching = strings.ReplaceAll(itemForMatching, "_", " ")

                // Collapse multiple spaces
                titleForMatching = strings.Join(strings.Fields(titleForMatching), " ")
                itemForMatching = strings.Join(strings.Fields(itemForMatching), " ")

                if !strings.Contains(titleForMatching, itemForMatching) {
                    log.Printf("PRE_FILTER_REJECTED: item phrase %q not found contiguously in title %q - skipping LLM\n",
                        itemForMatching, titleForMatching)
                    continue
                }

                log.Printf("PRE_FILTER_PASSED: item phrase %q found in title - proceeding to LLM\n", itemForMatching)

                // Extract entities from torrent title using LLM
                var entitiesJSON []byte = []byte("[]") // Initialize to empty JSON array
                var entities []Entity
                useEntityMatching := strings.ToLower(os.Getenv("USE_ENTITY_MATCHING")) == "true"

                if useEntityMatching {
                    log.Printf(">>> CALLING LLM for entity extraction: %q\n", r.Title)
                    entityResp, err := extractEntities(r.Title)
                    log.Printf("<<< LLM CALL COMPLETED for %q (error: %v)\n", r.Title, err)
                    if err != nil {
                        log.Printf("Entity extraction failed for %q: %v\n", r.Title, err)
                        // Fall back to fuzzy matching if entity extraction fails
                    } else {
                        entities = entityResp.Entities
                        entitiesJSON, _ = json.Marshal(entities)
                        log.Printf("Extracted %d entities from %q (URL: %s):\n", len(entities), r.Title, r.URL)
                        for i, entity := range entities {
                            log.Printf("  [%d] Type: %-20s Text: %-30s Confidence: %.2f\n",
                                i+1, entity.Type, entity.Text, entity.Confidence)
                        }
                    }
                }

                // Determine what to compare based on entity extraction
                var matched bool

                if useEntityMatching && len(entities) > 0 {
                    // Entity-based matching
                    filmTitleEntity := findEntityByType(entities, "FILM TITLE")
                    yearEntity := findEntityByType(entities, "YEAR")

                    if filmTitleEntity != nil {
                        // Compare item (without year) against FILM TITLE entity - EXACT MATCH REQUIRED
                        itemTitleLower := strings.ToLower(strings.TrimSpace(itemWithoutYear))
                        filmTitleLower := strings.ToLower(strings.TrimSpace(filmTitleEntity.Text))
                        exactMatch := itemTitleLower == filmTitleLower

                        log.Printf("EXACT_MATCH_CHECK item=%q (no year: %q) filmTitle=%q match=%v\n",
                            it.Text, itemWithoutYear, filmTitleEntity.Text, exactMatch)

                        if exactMatch {
                            // If item has a year, verify it matches
                            if itemYear != "" {
                                if yearEntity != nil && yearEntity.Text == itemYear {
                                    matched = true
                                    log.Printf("YEAR_MATCH item_year=%s entity_year=%s\n", itemYear, yearEntity.Text)
                                } else if yearEntity != nil {
                                    log.Printf("YEAR_MISMATCH item_year=%s entity_year=%s - REJECTED\n", itemYear, yearEntity.Text)
                                    continue
                                } else {
                                    log.Printf("NO_YEAR_ENTITY item_year=%s - REJECTED\n", itemYear)
                                    continue
                                }
                            } else {
                                // No year in item, just match on title
                                matched = true
                            }
                        } else {
                            log.Printf("TITLE_MISMATCH item=%q filmTitle=%q - REJECTED\n", itemWithoutYear, filmTitleEntity.Text)
                            continue // Skip fuzzy matching when entity matching explicitly rejects
                        }
                    } else {
                        log.Printf("NO_FILM_TITLE_ENTITY for %q - falling back to fuzzy\n", r.Title)
                    }
                }

                // Fall back to simple fuzzy matching if entity matching didn't work
                if !matched {
                    score := fuzzyScore(it.Text, r.Title)
                    log.Printf("FUZZY_SCORE=%.2f (threshold=%.2f) item=%q title=%q\n", score, threshold, it.Text, r.Title)
                    if score >= threshold {
                        matched = true
                    }
                }

                if !matched {
                    continue
                }

                // Match confirmed! Now extract magnet link from detail page
                log.Printf(">>> MATCH CONFIRMED for %q, extracting magnet link from %s\n", r.Title, r.URL)
                magnetLink, err := extractMagnetLinkFromURL(pw, r.URL)
                log.Printf("<<< MAGNET EXTRACTION COMPLETED for %s (error: %v)\n", r.URL, err)
                if err != nil {
                    log.Printf("Failed to extract magnet link from %s: %v\n", r.URL, err)
                    // Continue anyway - save match without magnet link
                    magnetLink = ""
                }

                log.Printf("Attempting to insert match with magnet_link=%q\n", magnetLink)
                inserted, err := insertMatchWithEntities(it.ID, r.Title, r.URL, s.Name(), r.Title, magnetLink, entitiesJSON)
                if err != nil {
                    log.Printf("insert match error: %v\n", err)
                    continue
                }
                if inserted {
                    matchesFound++
                    log.Printf("MATCH site=%s item=%q title=%q url=%s magnet=%q (match %d/%d)\n",
                        s.Name(), it.Text, r.Title, r.URL, magnetLink, matchesFound, maxMatchesPerItem)

                    // Broadcast new match via WebSocket
                    broadcastNewMatch(map[string]any{
                        "item":         it.Text,
                        "url":          r.URL,
                        "site":         s.Name(),
                        "torrent_text": r.Title,
                        "created":      time.Now().Format(time.RFC3339),
                    })

                    if err := maybeSendTwilioSMS(it.Text, r.Title, r.URL, s.Name()); err != nil {
                        log.Printf("twilio sms error: %v\n", err)
                    }

                    // Check if we've reached the limit after inserting
                    if matchesFound >= maxMatchesPerItem {
                        log.Printf("Reached %d matches for item %q, moving to next item\n", matchesFound, it.Text)
                        break
                    }
                }
            }
        }

        // Log completion of this item
        success := matchesFound > 0
        description := fmt.Sprintf("Item '%s' completed with %d match(es)", it.Text, matchesFound)
        if err := insertLog(description, success); err != nil {
            log.Printf("Failed to insert log for item %q: %v\n", it.Text, err)
        } else {
            log.Printf("LOG: %s (success=%v)\n", description, success)

            // Broadcast new log via WebSocket
            broadcastNewLog(map[string]any{
                "description": description,
                "success":     success,
                "timestamp":   time.Now().Format(time.RFC3339),
            })
        }
    }

    log.Println("Worker finished")
}

func loadItems() ([]Item, error) {
    rows, err := db.Query(`SELECT id, text FROM items ORDER BY id ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    out := make([]Item, 0, 64)
    for rows.Next() {
        var it Item
        if err := rows.Scan(&it.ID, &it.Text); err != nil {
            return nil, err
        }
        out = append(out, it)
    }
    return out, nil
}

func loadUrls() ([]URL, error) {
    rows, err := db.Query(`SELECT id, url, COALESCE(display_name, ''), COALESCE(config::text, '') FROM urls ORDER BY id ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    out := make([]URL, 0, 64)
    for rows.Next() {
        var u URL
        if err := rows.Scan(&u.ID, &u.URL, &u.DisplayName, &u.Config); err != nil {
            return nil, err
        }
        out = append(out, u)
    }
    return out, nil
}

// -------------------- Matching + quality rules --------------------

var nonAlnum = regexp.MustCompile(`[^a-z0-9\s]+`)
var multiSpace = regexp.MustCompile(`\s+`)

func normalize(s string) string {
    s = strings.ToLower(s)
    s = strings.ReplaceAll(s, "_", " ")
    s = strings.ReplaceAll(s, "-", " ")
    s = nonAlnum.ReplaceAllString(s, " ")
    s = multiSpace.ReplaceAllString(strings.TrimSpace(s), " ")
    return s
}

// fuzzyScore returns 0..1
func fuzzyScore(query, candidate string) float64 {
    q := normalize(query)
    c := normalize(candidate)
    if q == "" || c == "" {
        return 0
    }
    if q == c {
        return 1
    }

    // Token overlap
    qt := strings.Fields(q)
    ctset := map[string]struct{}{}
    for _, t := range strings.Fields(c) {
        ctset[t] = struct{}{}
    }
    hit := 0
    for _, t := range qt {
        if _, ok := ctset[t]; ok {
            hit++
        }
    }
    overlap := float64(hit) / float64(len(qt))

    // Fuzzy "contains-ish"
    contains := 0.0
    if fuzzy.Match(q, c) || strings.Contains(c, q) {
        contains = 1.0
    }

    // Blend, bias towards overlap
    score := 0.70*overlap + 0.30*contains
    return math.Max(0, math.Min(1, score))
}

// disqualifiedQuality checks for TS/CAM/TELECINE/Soundtrack tokens or "Telesync" substring (case-sensitive as requested)
func disqualifiedQuality(title string) bool {
    // TS, CAM, TELECINE, and Soundtrack must be standalone tokens (case-sensitive)
    // e.g. "Movie TS 1080p" -> disqualify
    // e.g. "Movie.TELECINE.avi" -> disqualify
    // e.g. "Movie (Original Motion Picture Soundtrack)" -> disqualify

    // Check for Soundtrack (case-insensitive)
    if strings.Contains(strings.ToLower(title), "soundtrack") {
        log.Printf("QUALITY_CHECK: Disqualified %q - contains 'soundtrack'\n", title)
        return true
    }

    // Check for Telesync
    if strings.Contains(title, "Telesync") {
        log.Printf("QUALITY_CHECK: Disqualified %q - contains 'Telesync'\n", title)
        return true
    }

    // Split by spaces first
    tokens := strings.Fields(title)
    for _, t := range tokens {
        // Also check tokens split by dots/dashes/underscores
        subTokens := strings.FieldsFunc(t, func(r rune) bool {
            return r == '.' || r == '-' || r == '_'
        })
        for _, st := range subTokens {
            stUpper := strings.ToUpper(st)
            if stUpper == "TS" || stUpper == "CAM" || stUpper == "TELECINE" || stUpper == "HDCAM" || stUpper == "CAMRIP" || stUpper == "HDTS" {
                log.Printf("QUALITY_CHECK: Disqualified %q - found token %q\n", title, st)
                return true
            }
        }
    }

    return false
}

// -------------------- Entity Extraction with Ollama --------------------

func extractYear(text string) string {
    // Match 4-digit year (1900-2099)
    re := regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
    matches := re.FindStringSubmatch(text)
    if len(matches) > 0 {
        return matches[0]
    }
    return ""
}

func removeYear(text string) string {
    // Remove 4-digit year and surrounding whitespace
    re := regexp.MustCompile(`\s*(19\d{2}|20\d{2})\s*`)
    return strings.TrimSpace(re.ReplaceAllString(text, " "))
}

func findEntityByType(entities []Entity, entityType string) *Entity {
    for _, e := range entities {
        if strings.Contains(strings.ToUpper(e.Type), entityType) {
            return &e
        }
    }
    return nil
}

func startOllama() error {
    ollamaURL := os.Getenv("OLLAMA_URL")
    if ollamaURL == "" {
        ollamaURL = "http://localhost:11434"
    }

    // Check if Ollama is already running
    resp, err := http.Get(ollamaURL + "/api/tags")
    if err == nil {
        resp.Body.Close()
        log.Println("Ollama is already running")
        return nil
    }

    // Ollama is not running, start it
    log.Println("Starting Ollama server...")
    ollamaCmd = exec.Command("ollama", "serve")

    // Start the process in the background
    if err := ollamaCmd.Start(); err != nil {
        return fmt.Errorf("failed to start Ollama: %w", err)
    }

    log.Printf("Ollama server started with PID %d", ollamaCmd.Process.Pid)

    // Wait for Ollama to be ready (max 30 seconds)
    for i := 0; i < 30; i++ {
        time.Sleep(1 * time.Second)
        resp, err := http.Get(ollamaURL + "/api/tags")
        if err == nil {
            resp.Body.Close()
            log.Println("Ollama server is ready")
            return nil
        }
    }

    return fmt.Errorf("Ollama server did not become ready within 30 seconds")
}

func checkOllamaHealth() error {
    ollamaURL := os.Getenv("OLLAMA_URL")
    if ollamaURL == "" {
        ollamaURL = "http://localhost:11434"
    }

    // Check if Ollama is running
    resp, err := http.Get(ollamaURL + "/api/tags")
    if err != nil {
        return fmt.Errorf("Ollama is not running at %s: %w", ollamaURL, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
    }

    // Check if the specified model is available
    ollamaModel := os.Getenv("OLLAMA_MODEL")
    if ollamaModel == "" {
        ollamaModel = "llama2"
    }

    var result struct {
        Models []struct {
            Name string `json:"name"`
        } `json:"models"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode Ollama models list: %w", err)
    }

    modelFound := false
    for _, m := range result.Models {
        if strings.HasPrefix(m.Name, ollamaModel) {
            modelFound = true
            break
        }
    }

    if !modelFound {
        return fmt.Errorf("model %q not found in Ollama. Run: ollama pull %s", ollamaModel, ollamaModel)
    }

    // Initialize the model by making a test generation call
    // This loads the model into memory if it's not already loaded
    log.Printf("Initializing model %q...", ollamaModel)
    testReq := OllamaRequest{
        Model:  ollamaModel,
        Prompt: "Hello",
        Stream: false,
    }

    jsonData, err := json.Marshal(testReq)
    if err != nil {
        return fmt.Errorf("failed to marshal test request: %w", err)
    }

    testResp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to initialize model: %w", err)
    }
    defer testResp.Body.Close()

    if testResp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(testResp.Body)
        return fmt.Errorf("model initialization failed with status %d: %s", testResp.StatusCode, string(body))
    }

    log.Printf("Model %q initialized successfully", ollamaModel)
    return nil
}

func extractEntities(text string) (*EntityExtractionResponse, error) {
    ollamaURL := os.Getenv("OLLAMA_URL")
    if ollamaURL == "" {
        ollamaURL = "http://localhost:11434" // Default Ollama URL
    }

    ollamaModel := os.Getenv("OLLAMA_MODEL")
    if ollamaModel == "" {
        ollamaModel = "llama2" // Default model
    }

    prompt := `Extract named entities from this torrent title and return ONLY a JSON object with an "entities" array. No explanations, no text, ONLY JSON.

Schema:
{
  "entities": [
    {
      "text": "string",
      "type": "FILM TITLE|YEAR|RESOLUTION|VIDEO FORMAT",
      "confidence": 0.95
    }
  ]
}

Torrent title: ` + text + `

JSON output:`

    reqBody := OllamaRequest{
        Model:  ollamaModel,
        Prompt: prompt,
        Stream: false,
        Format: "json", // Force JSON output
    }

    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    resp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to call Ollama: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
    }

    var ollamaResp OllamaResponse
    if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
        return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
    }

    // Check if response is empty or whitespace-only
    trimmedResponse := strings.TrimSpace(ollamaResp.Response)
    if trimmedResponse == "" {
        log.Printf("LLM returned empty response - treating as extraction failure")
        return nil, fmt.Errorf("LLM returned empty response")
    }

    // Parse the JSON response from the LLM
    // Try to parse as array first (LLM sometimes returns array directly)
    var entities []Entity
    if err := json.Unmarshal([]byte(ollamaResp.Response), &entities); err == nil {
        // Successfully parsed as array
        return &EntityExtractionResponse{Entities: entities}, nil
    }

    // Try to parse as object with "entities" field
    var entityResp EntityExtractionResponse
    if err := json.Unmarshal([]byte(ollamaResp.Response), &entityResp); err != nil {
        log.Printf("LLM returned invalid JSON - treating as extraction failure\nError: %v\nResponse: %q", err, ollamaResp.Response)
        return nil, fmt.Errorf("LLM returned invalid JSON: %w", err)
    }

    return &entityResp, nil
}

// -------------------- DB inserts + dedupe --------------------

func insertMatchDedup(itemID int64, matchedText, matchedURL, sourceSite, torrentText string) (bool, error) {
    // ON CONFLICT DO NOTHING provides dedupe via unique index (item_id, matched_url, source_site)
    res, err := db.Exec(`
        INSERT INTO matches(item_id, matched_text, matched_url, source_site, torrent_text)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (item_id, matched_url, source_site) DO NOTHING
    `, itemID, matchedText, matchedURL, sourceSite, torrentText)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func insertMatchWithEntities(itemID int64, matchedText, matchedURL, sourceSite, torrentText, magnetLink string, entitiesJSON []byte) (bool, error) {
    // Extract file size from entities
    var fileSize string
    var entities []Entity
    if err := json.Unmarshal(entitiesJSON, &entities); err == nil {
        for _, entity := range entities {
            if strings.ToUpper(entity.Type) == "FILE SIZE" || strings.ToUpper(entity.Type) == "FILESIZE" {
                fileSize = entity.Text
                break
            }
        }
    }

    // ON CONFLICT DO NOTHING provides dedupe via unique index (item_id, matched_url, source_site)
    res, err := db.Exec(`
        INSERT INTO matches(item_id, matched_text, matched_url, source_site, torrent_text, magnet_link, entities, file_size)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (item_id, matched_url, source_site) DO NOTHING
    `, itemID, matchedText, matchedURL, sourceSite, torrentText, magnetLink, entitiesJSON, fileSize)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func insertLog(description string, success bool) error {
    _, err := db.Exec(`
        INSERT INTO logs(description, success)
        VALUES ($1, $2)
    `, description, success)
    return err
}

// -------------------- Twilio (optional) --------------------

func maybeSendTwilioSMS(itemText, matchedTitle, matchedURL, site string) error {
    sid := strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID"))
    tok := strings.TrimSpace(os.Getenv("TWILIO_AUTH_TOKEN"))
    from := strings.TrimSpace(os.Getenv("TWILIO_FROM_NUMBER"))
    to := strings.TrimSpace(os.Getenv("ALERT_TO_NUMBER"))
    if sid == "" || tok == "" || from == "" || to == "" {
        return nil // not configured; do nothing
    }

    msg := fmt.Sprintf("Match found on %s\nItem: %s\nTitle: %s\n%s", site, itemText, matchedTitle, matchedURL)

    form := url.Values{}
    form.Set("From", from)
    form.Set("To", to)
    form.Set("Body", msg)

    req, err := http.NewRequest("POST",
        fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", sid),
        strings.NewReader(form.Encode()),
    )
    if err != nil {
        return err
    }
    req.SetBasicAuth(sid, tok)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{Timeout: 15 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("twilio status: %s", resp.Status)
    }
    return nil
}

// -------------------- API handlers --------------------

func itemsHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        rows, err := db.Query(`SELECT id, text FROM items ORDER BY created_at DESC`)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        out := make([]Item, 0, 64)
        for rows.Next() {
            var it Item
            if err := rows.Scan(&it.ID, &it.Text); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            out = append(out, it)
        }
        writeJSON(w, out)

    case http.MethodPost:
        if err := r.ParseForm(); err != nil {
            http.Error(w, "invalid form", http.StatusBadRequest)
            return
        }
        text := strings.TrimSpace(r.FormValue("text"))
        if text == "" {
            http.Error(w, "text required", http.StatusBadRequest)
            return
        }

        // Check if item already exists
        var existingID int64
        err := db.QueryRow(`SELECT id FROM items WHERE text = $1`, text).Scan(&existingID)
        if err == nil {
            // Item already exists
            w.WriteHeader(http.StatusConflict)
            writeJSON(w, map[string]any{"error": "Item already exists"})
            return
        } else if err != sql.ErrNoRows {
            // Database error
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        res, err := db.Exec(`INSERT INTO items(text) VALUES ($1)`, text)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        id, _ := res.LastInsertId()
        w.WriteHeader(http.StatusCreated)
        writeJSON(w, map[string]any{"id": id})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func itemHandler(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/items/")
    idStr = strings.Trim(idStr, "/")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil || id <= 0 {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    switch r.Method {
    case http.MethodPut:
        if err := r.ParseForm(); err != nil {
            http.Error(w, "invalid form", http.StatusBadRequest)
            return
        }
        text := strings.TrimSpace(r.FormValue("text"))
        if text == "" {
            http.Error(w, "text required", http.StatusBadRequest)
            return
        }
        _, err := db.Exec(`UPDATE items SET text=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`, text, id)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        writeJSON(w, map[string]any{"ok": true})

    case http.MethodDelete:
        _, err := db.Exec(`DELETE FROM items WHERE id=$1`, id)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        writeJSON(w, map[string]any{"ok": true})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func urlsHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        rows, err := db.Query(`SELECT id, url, COALESCE(display_name, ''), COALESCE(config::text, '') FROM urls ORDER BY id DESC`)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        out := make([]URL, 0, 64)
        for rows.Next() {
            var u URL
            if err := rows.Scan(&u.ID, &u.URL, &u.DisplayName, &u.Config); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            out = append(out, u)
        }
        writeJSON(w, out)

    case http.MethodPost:
        if err := r.ParseForm(); err != nil {
            http.Error(w, "invalid form", http.StatusBadRequest)
            return
        }
        urlStr := strings.TrimSpace(r.FormValue("url"))
        if urlStr == "" {
            http.Error(w, "url required", http.StatusBadRequest)
            return
        }
        displayName := strings.TrimSpace(r.FormValue("display_name"))
        configStr := strings.TrimSpace(r.FormValue("config"))
        var res sql.Result
        var err error

        if configStr != "" && displayName != "" {
            res, err = db.Exec(`INSERT INTO urls(url, display_name, config) VALUES ($1, $2, $3::jsonb)`, urlStr, displayName, configStr)
        } else if configStr != "" {
            res, err = db.Exec(`INSERT INTO urls(url, config) VALUES ($1, $2::jsonb)`, urlStr, configStr)
        } else if displayName != "" {
            res, err = db.Exec(`INSERT INTO urls(url, display_name) VALUES ($1, $2)`, urlStr, displayName)
        } else {
            res, err = db.Exec(`INSERT INTO urls(url) VALUES ($1)`, urlStr)
        }
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        id, _ := res.LastInsertId()
        w.WriteHeader(http.StatusCreated)
        writeJSON(w, map[string]any{"id": id})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func urlHandler(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/urls/")
    idStr = strings.Trim(idStr, "/")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil || id <= 0 {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    switch r.Method {
    case http.MethodPut:
        if err := r.ParseForm(); err != nil {
            http.Error(w, "invalid form", http.StatusBadRequest)
            return
        }
        urlStr := strings.TrimSpace(r.FormValue("url"))
        displayName := strings.TrimSpace(r.FormValue("display_name"))
        configStr := strings.TrimSpace(r.FormValue("config"))

        if urlStr == "" && displayName == "" && configStr == "" {
            http.Error(w, "url, display_name, or config required", http.StatusBadRequest)
            return
        }

        // Build dynamic update query
        updates := []string{}
        args := []interface{}{}
        argPos := 1

        if urlStr != "" {
            updates = append(updates, fmt.Sprintf("url=$%d", argPos))
            args = append(args, urlStr)
            argPos++
        }
        if displayName != "" {
            updates = append(updates, fmt.Sprintf("display_name=$%d", argPos))
            args = append(args, displayName)
            argPos++
        }
        if configStr != "" {
            updates = append(updates, fmt.Sprintf("config=$%d::jsonb", argPos))
            args = append(args, configStr)
            argPos++
        }

        updates = append(updates, "updated_at=CURRENT_TIMESTAMP")
        args = append(args, id)

        query := fmt.Sprintf("UPDATE urls SET %s WHERE id=$%d", strings.Join(updates, ", "), argPos)
        _, err = db.Exec(query, args...)

        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        writeJSON(w, map[string]any{"ok": true})

    case http.MethodDelete:
        _, err := db.Exec(`DELETE FROM urls WHERE id=$1`, id)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        writeJSON(w, map[string]any{"ok": true})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func matchesHandler(w http.ResponseWriter, r *http.Request) {
    if r.URL.RawQuery != "" {
        log.Printf("GET /api/matches - Query params: %s", r.URL.RawQuery)
    } else {
        log.Printf("GET /api/matches - No query params")
    }

    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    rows, err := db.Query(`
        SELECT m.id, i.text, m.matched_url, m.source_site, COALESCE(m.torrent_text, ''), COALESCE(m.magnet_link, ''), COALESCE(m.file_size, ''), m.created_at
        FROM matches m
        JOIN items i ON i.id = m.item_id
        ORDER BY m.created_at DESC
        LIMIT 200
    `)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    type Match struct {
        ID          int64  `json:"id"`
        Item        string `json:"item"`
        URL         string `json:"url"`
        Site        string `json:"site"`
        TorrentText string `json:"torrent_text,omitempty"`
        MagnetLink  string `json:"magnet_link,omitempty"`
        FileSize    string `json:"file_size,omitempty"`
        Created     string `json:"created"`
    }
    out := make([]Match, 0, 200)
    for rows.Next() {
        var m Match
        if err := rows.Scan(&m.ID, &m.Item, &m.URL, &m.Site, &m.TorrentText, &m.MagnetLink, &m.FileSize, &m.Created); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        out = append(out, m)
    }
    writeJSON(w, out)
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        // Parse pagination parameters
        pageStr := r.URL.Query().Get("page")
        page := 1
        if pageStr != "" {
            if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
                page = p
            }
        }

        pageSize := 25
        offset := (page - 1) * pageSize

        // Get total count
        var total int
        err := db.QueryRow(`SELECT COUNT(*) FROM logs`).Scan(&total)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        // Get paginated logs
        rows, err := db.Query(`
            SELECT id, timestamp, description, success
            FROM logs
            ORDER BY timestamp DESC
            LIMIT $1 OFFSET $2
        `, pageSize, offset)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        type Log struct {
            ID          int64  `json:"id"`
            Timestamp   string `json:"timestamp"`
            Description string `json:"description"`
            Success     bool   `json:"success"`
        }

        logs := make([]Log, 0, pageSize)
        for rows.Next() {
            var l Log
            if err := rows.Scan(&l.ID, &l.Timestamp, &l.Description, &l.Success); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            logs = append(logs, l)
        }

        writeJSON(w, map[string]any{
            "logs":       logs,
            "page":       page,
            "page_size":  pageSize,
            "total":      total,
            "total_pages": (total + pageSize - 1) / pageSize,
        })

    case http.MethodDelete:
        // Delete all logs
        result, err := db.Exec(`DELETE FROM logs`)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        rowsAffected, _ := result.RowsAffected()
        log.Printf("Cleared %d log entries", rowsAffected)

        writeJSON(w, map[string]any{
            "ok":      true,
            "deleted": rowsAffected,
        })

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func matchHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    idStr := strings.TrimPrefix(r.URL.Path, "/api/matches/")
    log.Printf("DELETE /api/matches/%s - Attempting to delete match ID: %s\n", idStr, idStr)

    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        log.Printf("DELETE /api/matches/%s - Invalid ID format: %v\n", idStr, err)
        http.Error(w, "Invalid match ID", http.StatusBadRequest)
        return
    }

    result, err := db.Exec("DELETE FROM matches WHERE id=$1", id)
    if err != nil {
        log.Printf("DELETE /api/matches/%d - Database error: %v\n", id, err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := result.RowsAffected()
    log.Printf("DELETE /api/matches/%d - Successfully deleted %d row(s)\n", id, rowsAffected)

    writeJSON(w, map[string]any{"ok": true})
}

func workerStatusHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    writeJSON(w, map[string]any{
        "running": workerRunning.Load(),
    })
}

func triggerWorkerHandler(w http.ResponseWriter, r *http.Request) {
    if r.URL.RawQuery != "" {
        log.Printf("POST /api/trigger-worker - Query params: %s", r.URL.RawQuery)
    } else {
        log.Printf("POST /api/trigger-worker - No query params")
    }

    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    if workerRunning.Load() {
        writeJSON(w, map[string]any{
            "status":  "already_running",
            "message": "Worker is already running",
        })
        return
    }

    go runWorker()

    writeJSON(w, map[string]any{
        "status":  "triggered",
        "message": "Worker triggered successfully",
    })
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := wsUpgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade error: %v", err)
        return
    }

    wsClientsMux.Lock()
    wsClients[conn] = true
    wsClientsMux.Unlock()

    log.Printf("WebSocket client connected. Total clients: %d", len(wsClients))

    // Keep connection alive and handle disconnection
    defer func() {
        wsClientsMux.Lock()
        delete(wsClients, conn)
        wsClientsMux.Unlock()
        conn.Close()
        log.Printf("WebSocket client disconnected. Total clients: %d", len(wsClients))
    }()

    // Read messages from client (ping/pong to keep alive)
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            break
        }
    }
}

func broadcastWorkerStatus(status string, message string) {
    wsClientsMux.Lock()
    defer wsClientsMux.Unlock()

    msg := map[string]any{
        "type":    "worker_status",
        "status":  status,
        "message": message,
    }

    for client := range wsClients {
        if err := client.WriteJSON(msg); err != nil {
            log.Printf("WebSocket write error: %v", err)
            client.Close()
            delete(wsClients, client)
        }
    }
}

func broadcastNewMatch(match map[string]any) {
    wsClientsMux.Lock()
    defer wsClientsMux.Unlock()

    msg := map[string]any{
        "type":  "new_match",
        "match": match,
    }

    for client := range wsClients {
        if err := client.WriteJSON(msg); err != nil {
            log.Printf("WebSocket write error: %v", err)
            client.Close()
            delete(wsClients, client)
        }
    }
}

func broadcastNewLog(logEntry map[string]any) {
    wsClientsMux.Lock()
    defer wsClientsMux.Unlock()

    msg := map[string]any{
        "type": "new_log",
        "log":  logEntry,
    }

    for client := range wsClients {
        if err := client.WriteJSON(msg); err != nil {
            log.Printf("WebSocket write error: %v", err)
            client.Close()
            delete(wsClients, client)
        }
    }
}

func writeJSON(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json")
    enc := json.NewEncoder(w)
    enc.SetEscapeHTML(true)
    _ = enc.Encode(v)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
    })
}

// -------------------- DB init --------------------

func initDB(db *sql.DB) error {
    stmts := []string{
        `CREATE TABLE IF NOT EXISTS items (
            id SERIAL PRIMARY KEY,
            text TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP
        );`,

        `CREATE TABLE IF NOT EXISTS matches (
            id SERIAL PRIMARY KEY,
            item_id INTEGER NOT NULL,
            matched_text TEXT,
            matched_url TEXT NOT NULL,
            source_site TEXT NOT NULL,
            torrent_text VARCHAR(500),
            magnet_link TEXT,
            entities JSONB,
            file_size VARCHAR(50),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE
        );`,

        `CREATE UNIQUE INDEX IF NOT EXISTS ux_matches_dedupe
         ON matches(item_id, matched_url, source_site);`,

        `CREATE TABLE IF NOT EXISTS urls (
            id SERIAL PRIMARY KEY,
            url TEXT NOT NULL UNIQUE,
            display_name TEXT,
            config JSONB,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP
        );`,

        `CREATE TABLE IF NOT EXISTS logs (
            id SERIAL PRIMARY KEY,
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            description TEXT NOT NULL,
            success BOOLEAN NOT NULL
        );`,

        // Add config column if it doesn't exist (for existing tables)
        `DO $$ 
        BEGIN 
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns 
                WHERE table_name='urls' AND column_name='config'
            ) THEN
                ALTER TABLE urls ADD COLUMN config JSONB;
            END IF;
        END $$;`,

        // Add display_name column if it doesn't exist (for existing tables)
        `DO $$ 
        BEGIN 
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns 
                WHERE table_name='urls' AND column_name='display_name'
            ) THEN
                ALTER TABLE urls ADD COLUMN display_name TEXT;
            END IF;
        END $$;`,

        // Add torrent_text column if it doesn't exist (for existing tables)
        `DO $$ 
        BEGIN 
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns 
                WHERE table_name='matches' AND column_name='torrent_text'
            ) THEN
                ALTER TABLE matches ADD COLUMN torrent_text VARCHAR(500);
            END IF;
        END $$;`,

        // Add entities column if it doesn't exist (for existing tables)
        `DO $$ 
        BEGIN 
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns 
                WHERE table_name='matches' AND column_name='entities'
            ) THEN
                ALTER TABLE matches ADD COLUMN entities JSONB;
            END IF;
        END $$;`,

        // Add magnet_link column if it doesn't exist
        `DO $$ 
        BEGIN 
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns 
                WHERE table_name = 'matches' AND column_name = 'magnet_link'
            ) THEN
                ALTER TABLE matches ADD COLUMN magnet_link TEXT;
            END IF;
        END $$;`,

        // Update foreign key constraint to include ON DELETE CASCADE
        `DO $$ 
        BEGIN 
            -- Drop the old constraint if it exists
            IF EXISTS (
                SELECT 1 FROM information_schema.table_constraints 
                WHERE constraint_name = 'matches_item_id_fkey' 
                AND table_name = 'matches'
            ) THEN
                ALTER TABLE matches DROP CONSTRAINT matches_item_id_fkey;
                ALTER TABLE matches ADD CONSTRAINT matches_item_id_fkey 
                    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE;
            END IF;
        END $$;`,
    }

    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return err
        }
    }
    return nil
}

// -------------------- Generic URL scraper --------------------

type GenericScraper struct {
    URL         string
    DisplayName string
    Config      string // JSON config with selectors
}

func (s *GenericScraper) Name() string { return s.DisplayName }

func (s *GenericScraper) Search(ctx context.Context, pw *playwright.Playwright, query string) ([]SearchResult, error) {
    // If Playwright is disabled, just return no results.
    if pw == nil {
        return nil, nil
    }

    browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
        Headless: playwright.Bool(true),
    })
    if err != nil {
        return nil, err
    }
    defer func() { _ = browser.Close() }()

    page, err := browser.NewPage()
    if err != nil {
        return nil, err
    }
    defer func() { _ = page.Close() }()

    log.Printf("Navigating to %s to search for %q\n", s.URL, query)

    // Navigate to the base URL first
    if _, err := page.Goto(s.URL, playwright.PageGotoOptions{
        WaitUntil: playwright.WaitUntilStateNetworkidle,
        Timeout:   playwright.Float(30000),
    }); err != nil {
        return nil, err
    }

    // Parse config for search box and button selectors
    var config map[string]interface{}
    searchInputSelector := "input[type='search'], input[name='q'], input[name='query'], input[name='search']"
    searchButtonSelector := "button[type='submit'], input[type='submit'], button:has-text('Search')"

    if s.Config != "" {
        if err := json.Unmarshal([]byte(s.Config), &config); err != nil {
            log.Printf("Failed to parse config: %v\n", err)
        } else {
            if sel, ok := config["searchInputSelector"].(string); ok && sel != "" {
                searchInputSelector = sel
            }
            if sel, ok := config["searchButtonSelector"].(string); ok && sel != "" {
                searchButtonSelector = sel
            }
        }
    }

    log.Printf("Looking for search input with selector: %s\n", searchInputSelector)

    // Find and fill the search input
    searchInput := page.Locator(searchInputSelector).First()
    if err := searchInput.WaitFor(playwright.LocatorWaitForOptions{
        State:   playwright.WaitForSelectorStateVisible,
        Timeout: playwright.Float(10000),
    }); err != nil {
        log.Printf("Could not find search input: %v\n", err)
        return nil, fmt.Errorf("search input not found")
    }

    if err := searchInput.Fill(query); err != nil {
        log.Printf("Failed to fill search input: %v\n", err)
        return nil, err
    }

    log.Printf("Filled search input, looking for search button: %s\n", searchButtonSelector)

    // Click the search button with timeout
    searchButton := page.Locator(searchButtonSelector).First()

    // Wait for button to be visible with timeout
    if err := searchButton.WaitFor(playwright.LocatorWaitForOptions{
        State:   playwright.WaitForSelectorStateVisible,
        Timeout: playwright.Float(5000),
    }); err != nil {
        log.Printf("Search button not found, trying Enter key: %v\n", err)
        if err := searchInput.Press("Enter"); err != nil {
            log.Printf("Failed to press Enter: %v\n", err)
            return nil, fmt.Errorf("could not submit search")
        }
    } else {
        if err := searchButton.Click(playwright.LocatorClickOptions{
            Timeout: playwright.Float(5000),
        }); err != nil {
            log.Printf("Failed to click search button, trying Enter: %v\n", err)
            if err := searchInput.Press("Enter"); err != nil {
                log.Printf("Failed to press Enter: %v\n", err)
                return nil, fmt.Errorf("could not submit search")
            }
        }
    }

    log.Printf("Search submitted, waiting for navigation and results...\n")

    // Wait for navigation/results to load with shorter timeout
    time.Sleep(2 * time.Second) // Give page time to start loading
    if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
        State:   playwright.LoadStateNetworkidle,
        Timeout: playwright.Float(15000),
    }); err != nil {
        log.Printf("Warning: page load state timeout: %v (continuing anyway)\n", err)
        // Continue anyway, results might be loaded
    }

    // Log the current URL to confirm navigation
    currentURL := page.URL()
    log.Printf("Search completed, now on page: %s\n", currentURL)

    // Save screenshot and HTML for debugging/config creation
    timestamp := time.Now().Unix()
    screenshotPath := fmt.Sprintf("data/screenshots/%s_%d.png", url.QueryEscape(s.URL), timestamp)
    htmlPath := fmt.Sprintf("data/html/%s_%d.html", url.QueryEscape(s.URL), timestamp)

    // Create directories if they don't exist
    os.MkdirAll("data/screenshots", 0755)
    os.MkdirAll("data/html", 0755)

    // Save screenshot
    if _, err := page.Screenshot(playwright.PageScreenshotOptions{
        Path: playwright.String(screenshotPath),
        FullPage: playwright.Bool(true),
    }); err != nil {
        log.Printf("Failed to save screenshot: %v\n", err)
    } else {
        log.Printf("Saved screenshot: %s\n", screenshotPath)
    }

    // Save HTML
    htmlContent, err := page.Content()
    if err != nil {
        log.Printf("Failed to get page content: %v\n", err)
    } else {
        if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
            log.Printf("Failed to save HTML: %v\n", err)
        } else {
            log.Printf("Saved HTML: %s\n", htmlPath)
        }
    }

    // Parse config for link selector and extraction sequence
    var linkSelector string = "a" // default
    var extractionSteps []interface{}

    if s.Config != "" && config != nil {
        // Check if config has a link selector
        if sel, ok := config["linkSelector"].(string); ok && sel != "" {
            linkSelector = sel
            log.Printf("Using custom link selector: %s\n", linkSelector)
        }
        // Check if config has extraction steps
        if steps, ok := config["extractionSteps"].([]interface{}); ok {
            extractionSteps = steps
            log.Printf("Using %d extraction steps from config\n", len(extractionSteps))
        }
    }

    // Try to find torrent links using config selector or default
    links, err := page.Locator(linkSelector).All()
    if err != nil {
        return nil, err
    }

    // First pass: Extract all link data from search results page (cache it)
    type LinkData struct {
        Href string
        Text string
    }
    var linkDataList []LinkData
    seen := make(map[string]bool)

    log.Printf("Extracting link data from search results page...\n")
    for _, link := range links {
        href, err := link.GetAttribute("href")
        if err != nil || href == "" {
            continue
        }

        text, err := link.InnerText()
        if err != nil || text == "" {
            continue
        }

        text = strings.TrimSpace(text)

        // Skip navigation/short links - torrent titles are usually longer
        if len(text) < 10 {
            continue
        }

        // Skip common navigation text
        lowerText := strings.ToLower(text)
        if lowerText == "home" || lowerText == "login" || lowerText == "register" ||
           lowerText == "about" || lowerText == "contact" || lowerText == "privacy" ||
           lowerText == "terms of service" || lowerText == "dmca" || strings.HasPrefix(lowerText, "page ") {
            continue
        }

        // Make sure href is absolute
        if !strings.HasPrefix(href, "http") {
            if strings.HasPrefix(href, "/") {
                parsedBase, _ := url.Parse(s.URL)
                href = parsedBase.Scheme + "://" + parsedBase.Host + href
            } else {
                continue
            }
        }

        // Validate URL can be parsed
        parsedURL, err := url.Parse(href)
        if err != nil {
            log.Printf("Skipping malformed URL for %q: %v\n", text, err)
            continue
        }

        // Use the parsed URL's string representation (properly encoded)
        href = parsedURL.String()

        // Skip if we've seen this URL
        if seen[href] {
            continue
        }

        log.Printf("Cached link: title=%q url=%s\n", text, href)
        seen[href] = true
        linkDataList = append(linkDataList, LinkData{Href: href, Text: text})
    }

    log.Printf("Cached %d links from search results page (will process until matches found)\n", len(linkDataList))

    // Second pass: Process cached links and extract magnet links from detail pages
    results := []SearchResult{}

    // Just return cached search results WITHOUT magnet links
    // Worker will extract magnet links only for confirmed matches
    for _, linkData := range linkDataList {
        results = append(results, SearchResult{
            Title:      linkData.Text,
            URL:        linkData.Href,
            MagnetLink: "", // Will be extracted later for matches only
        })
    }

    log.Printf("Found %d potential results from %s\n", len(results), s.URL)
    return results, nil
}

// extractMagnetLink extracts magnet link from a torrent detail page
func extractMagnetLinkFromURL(pw *playwright.Playwright, detailURL string) (string, error) {
    if pw == nil {
        return "", fmt.Errorf("playwright not available")
    }

    // Parse and properly encode the URL to handle non-ASCII characters
    parsedURL, err := url.Parse(detailURL)
    if err != nil {
        log.Printf("Failed to parse URL %q: %v\n", detailURL, err)
        return "", fmt.Errorf("invalid URL: %v", err)
    }

    // Get the properly encoded URL string
    encodedURL := parsedURL.String()
    log.Printf("Navigating to detail page (encoded): %s\n", encodedURL)

    browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
        Headless: playwright.Bool(true),
    })
    if err != nil {
        return "", err
    }
    defer browser.Close()

    page, err := browser.NewPage()
    if err != nil {
        return "", err
    }
    defer page.Close()

    // Navigate to detail page using encoded URL
    if _, err := page.Goto(encodedURL, playwright.PageGotoOptions{
        WaitUntil: playwright.WaitUntilStateNetworkidle,
        Timeout:   playwright.Float(10000),
    }); err != nil {
        return "", fmt.Errorf("navigation failed: %v", err)
    }

    // Save screenshot and HTML for debugging
    timestamp := time.Now().Unix()
    screenshotPath := fmt.Sprintf("data/screenshots/magnet_%s_%d.png", url.QueryEscape(detailURL), timestamp)
    htmlPath := fmt.Sprintf("data/html/magnet_%s_%d.html", url.QueryEscape(detailURL), timestamp)

    os.MkdirAll("data/screenshots", 0755)
    os.MkdirAll("data/html", 0755)

    page.Screenshot(playwright.PageScreenshotOptions{
        Path: playwright.String(screenshotPath),
        FullPage: playwright.Bool(true),
    })

    htmlContent, _ := page.Content()
    os.WriteFile(htmlPath, []byte(htmlContent), 0644)

    log.Printf("Saved magnet extraction debug files: %s, %s\n", screenshotPath, htmlPath)

    // Look for magnet link - try direct magnet links first
    magnetLocator := page.Locator("a:has-text('Magnet Link'), a:has-text('Magnet Download'), a[href^='magnet:']").First()
    magnetLink, err := magnetLocator.GetAttribute("href")
    
    // If direct magnet link found, return it
    if err == nil && magnetLink != "" && strings.HasPrefix(magnetLink, "magnet:") {
        log.Printf("Extracted direct magnet link from %s: %s\n", detailURL, magnetLink)
        return magnetLink, nil
    }
    
    // For BT4G and similar sites, look for keepshare.org links that contain encoded magnet links
    keepshareLocator := page.Locator("a[href*='keepshare.org']").First()
    keepshareURL, err := keepshareLocator.GetAttribute("href")
    if err == nil && keepshareURL != "" {
        // The magnet link is URL-encoded in the keepshare URL path
        // Format: //keepshare.org/16b6v173/magnet:%3Fxt=urn:btih:...
        if strings.Contains(keepshareURL, "magnet:") {
            // Extract and decode the magnet link
            parts := strings.Split(keepshareURL, "/magnet:")
            if len(parts) >= 2 {
                encodedMagnet := "magnet:" + parts[1]
                // URL decode it
                decodedMagnet, err := url.QueryUnescape(encodedMagnet)
                if err == nil && decodedMagnet != "" {
                    log.Printf("Extracted magnet link from keepshare URL %s: %s\n", detailURL, decodedMagnet)
                    return decodedMagnet, nil
                }
            }
        }
    }
    
    // Try looking for any link with magnet: in href as last resort
    anyMagnetLocator := page.Locator("a[href*='magnet:']").First()
    anyMagnetLink, err := anyMagnetLocator.GetAttribute("href")
    if err == nil && anyMagnetLink != "" {
        if strings.HasPrefix(anyMagnetLink, "magnet:") {
            log.Printf("Extracted magnet link (fallback) from %s: %s\n", detailURL, anyMagnetLink)
            return anyMagnetLink, nil
        }
        // Try to extract magnet from URL-encoded link
        if strings.Contains(anyMagnetLink, "magnet:") || strings.Contains(anyMagnetLink, "magnet%3A") {
            decoded, _ := url.QueryUnescape(anyMagnetLink)
            if strings.Contains(decoded, "magnet:") {
                magnetStart := strings.Index(decoded, "magnet:")
                magnetPart := decoded[magnetStart:]
                log.Printf("Extracted decoded magnet link from %s: %s\n", detailURL, magnetPart)
                return magnetPart, nil
            }
        }
    }

    return "", fmt.Errorf("magnet link not found")
}

// extractTorrentURL follows config-driven steps to extract the actual torrent URL
func (s *GenericScraper) extractTorrentURL(browser playwright.Browser, startURL string, steps []interface{}) (string, error) {
    page, err := browser.NewPage()
    if err != nil {
        return "", err
    }
    defer func() { _ = page.Close() }()

    currentURL := startURL
    log.Printf("Starting extraction from %s\n", startURL)

    // Navigate to the initial URL
    if _, err := page.Goto(currentURL, playwright.PageGotoOptions{
        WaitUntil: playwright.WaitUntilStateNetworkidle,
        Timeout:   playwright.Float(15000),
    }); err != nil {
        return "", fmt.Errorf("failed to navigate to %s: %w", currentURL, err)
    }

    // Execute each step in sequence
    for i, stepInterface := range steps {
        stepMap, ok := stepInterface.(map[string]interface{})
        if !ok {
            log.Printf("Step %d: invalid step format\n", i)
            continue
        }

        action, _ := stepMap["action"].(string)
        selector, _ := stepMap["selector"].(string)
        attribute, _ := stepMap["attribute"].(string)

        log.Printf("Step %d: action=%s selector=%s\n", i, action, selector)

        switch action {
        case "click":
            // Click on an element
            elem := page.Locator(selector).First()
            if err := elem.WaitFor(playwright.LocatorWaitForOptions{
                State:   playwright.WaitForSelectorStateVisible,
                Timeout: playwright.Float(10000),
            }); err != nil {
                return "", fmt.Errorf("step %d: element not found: %s", i, selector)
            }
            
            if err := elem.Click(); err != nil {
                return "", fmt.Errorf("step %d: click failed: %w", i, err)
            }

            // Wait for navigation if needed
            time.Sleep(1 * time.Second)
            if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
                State:   playwright.LoadStateNetworkidle,
                Timeout: playwright.Float(10000),
            }); err != nil {
                log.Printf("Step %d: load state timeout (continuing anyway)\n", i)
            }

        case "clickNewPage":
            // Click on an element that opens a new page/tab
            elem := page.Locator(selector).First()
            if err := elem.WaitFor(playwright.LocatorWaitForOptions{
                State:   playwright.WaitForSelectorStateVisible,
                Timeout: playwright.Float(10000),
            }); err != nil {
                return "", fmt.Errorf("step %d: element not found: %s", i, selector)
            }

            // Listen for new page
            newPage, err := page.Context().ExpectPage(func() error {
                return elem.Click()
            })
            
            if err != nil {
                return "", fmt.Errorf("step %d: failed to open new page: %w", i, err)
            }

            // Switch to new page
            _ = page.Close()
            page = newPage
            
            if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
                State:   playwright.LoadStateNetworkidle,
                Timeout: playwright.Float(10000),
            }); err != nil {
                log.Printf("Step %d: new page load timeout (continuing anyway)\n", i)
            }

        case "extract":
            // Extract attribute from an element
            elem := page.Locator(selector).First()
            if err := elem.WaitFor(playwright.LocatorWaitForOptions{
                State:   playwright.WaitForSelectorStateVisible,
                Timeout: playwright.Float(10000),
            }); err != nil {
                return "", fmt.Errorf("step %d: element not found: %s", i, selector)
            }

            var extractedValue string
            if attribute == "text" {
                extractedValue, err = elem.InnerText()
            } else {
                extractedValue, err = elem.GetAttribute(attribute)
            }
            
            if err != nil || extractedValue == "" {
                return "", fmt.Errorf("step %d: failed to extract %s", i, attribute)
            }

            log.Printf("Extracted torrent URL: %s\n", extractedValue)
            return extractedValue, nil
        }
    }

    return "", fmt.Errorf("no extraction step found in config")
}

// -------------------- Example.com scraper --------------------

type ExampleComScraper struct{}

func (s *ExampleComScraper) Name() string { return "example.com" }

func (s *ExampleComScraper) Search(ctx context.Context, pw *playwright.Playwright, query string) ([]SearchResult, error) {
    // If Playwright is disabled, just return no results.
    if pw == nil {
        return nil, nil
    }

    browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
        Headless: playwright.Bool(true),
    })
    if err != nil {
        return nil, err
    }
    defer func() { _ = browser.Close() }()

    page, err := browser.NewPage()
    if err != nil {
        return nil, err
    }
    defer func() { _ = page.Close() }()

    if _, err := page.Goto("https://www.example.com", playwright.PageGotoOptions{
        WaitUntil: playwright.WaitUntilStateNetworkidle,
        Timeout:   playwright.Float(30000),
    }); err != nil {
        return nil, err
    }

    title, err := page.Title()
    if err != nil {
        return nil, err
    }

    // This adapter just returns the Example Domain page title as a single "result".
    // Use it to validate that fuzzy matching + dedupe + SMS wiring works.
    return []SearchResult{{
        Title: title,
        URL:   "https://www.example.com/",
    }}, nil
}
