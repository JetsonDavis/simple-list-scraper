package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "math"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "regexp"
    "strconv"
    "strings"
    "sync/atomic"
    "syscall"
    "time"

    "github.com/lithammer/fuzzysearch/fuzzy"
    _ "github.com/mattn/go-sqlite3"
    "github.com/playwright-community/playwright-go"
)

var (
    db            *sql.DB
    workerRunning atomic.Bool
)

type Item struct {
    ID   int64  `json:"id"`
    Text string `json:"text"`
}

type SearchResult struct {
    Title string
    URL   string
}

type SiteScraper interface {
    Name() string
    Search(ctx context.Context, pw *playwright.Playwright, query string) ([]SearchResult, error)
}

func main() {
    dbPath := os.Getenv("DB_PATH")
    if dbPath == "" {
        log.Fatal("DB_PATH not set (e.g. /data/app.db)")
    }

    intervalHours := getenvInt("CHECK_INTERVAL_HOURS", 6)
    if intervalHours <= 0 {
        intervalHours = 6
    }
    interval := time.Duration(intervalHours) * time.Hour

    runOnStart := strings.ToLower(os.Getenv("RUN_WORKER_ON_START"))
    if runOnStart == "" {
        runOnStart = "true"
    }

    var err error
    db, err = sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatal(err)
    }
    if err := initDB(db); err != nil {
        log.Fatal(err)
    }

    go scheduler(interval, runOnStart == "true")

    mux := http.NewServeMux()
    mux.HandleFunc("/api/items", itemsHandler)
    mux.HandleFunc("/api/items/", itemHandler)
    mux.HandleFunc("/api/matches", matchesHandler)
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
    defer workerRunning.Store(false)

    disablePW := strings.ToLower(os.Getenv("DISABLE_PLAYWRIGHT")) == "true"
    threshold := getenvFloat("FUZZY_THRESHOLD", 0.78)

    log.Printf("Worker started (threshold=%.2f, playwright_disabled=%v)\n", threshold, disablePW)

    items, err := loadItems()
    if err != nil {
        log.Println("worker load items:", err)
        return
    }
    if len(items) == 0 {
        log.Println("worker: no items; done")
        return
    }

    scrapers := []SiteScraper{
        &ExampleComScraper{},
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
        for _, s := range scrapers {
            results, err := s.Search(context.Background(), pw, it.Text)
            if err != nil {
                log.Printf("scraper %s error: %v\n", s.Name(), err)
                continue
            }
            for _, r := range results {
                if disqualifiedQuality(r.Title) {
                    log.Printf("DISQUALIFIED_QUALITY site=%s url=%s title=%q\n", s.Name(), r.URL, r.Title)
                    continue
                }

                score := fuzzyScore(it.Text, r.Title)
                if score < threshold {
                    continue
                }

                inserted, err := insertMatchDedup(it.ID, r.Title, r.URL, s.Name())
                if err != nil {
                    log.Printf("insert match error: %v\n", err)
                    continue
                }
                if inserted {
                    log.Printf("MATCH site=%s score=%.2f item=%q title=%q url=%s\n",
                        s.Name(), score, it.Text, r.Title, r.URL)

                    if err := maybeSendTwilioSMS(it.Text, r.Title, r.URL, s.Name()); err != nil {
                        log.Printf("twilio sms error: %v\n", err)
                    }
                }
            }
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

// disqualifiedQuality checks for TS/CAM tokens or "Telesync" substring (case-sensitive as requested)
func disqualifiedQuality(title string) bool {
    // TS and CAM must be standalone tokens (case-sensitive)
    // e.g. "Movie TS 1080p" -> disqualify
    tokens := strings.Fields(title)
    for _, t := range tokens {
        if t == "TS" || t == "CAM" {
            return true
        }
    }
    if strings.Contains(title, "Telesync") {
        return true
    }
    return false
}

// -------------------- DB inserts + dedupe --------------------

func insertMatchDedup(itemID int64, matchedText, matchedURL, sourceSite string) (bool, error) {
    // INSERT OR IGNORE provides dedupe via unique index (item_id, matched_url, source_site)
    res, err := db.Exec(`
        INSERT OR IGNORE INTO matches(item_id, matched_text, matched_url, source_site)
        VALUES (?, ?, ?, ?)
    `, itemID, matchedText, matchedURL, sourceSite)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
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
        rows, err := db.Query(`SELECT id, text FROM items ORDER BY id DESC`)
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
        res, err := db.Exec(`INSERT INTO items(text) VALUES (?)`, text)
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
        _, err := db.Exec(`UPDATE items SET text=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, text, id)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        writeJSON(w, map[string]any{"ok": true})

    case http.MethodDelete:
        _, err := db.Exec(`DELETE FROM items WHERE id=?`, id)
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
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    rows, err := db.Query(`
        SELECT m.id, i.text, m.matched_url, m.source_site, m.created_at
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
        ID      int64  `json:"id"`
        Item    string `json:"item"`
        URL     string `json:"url"`
        Site    string `json:"site"`
        Created string `json:"created"`
    }
    out := make([]Match, 0, 200)
    for rows.Next() {
        var m Match
        if err := rows.Scan(&m.ID, &m.Item, &m.URL, &m.Site, &m.Created); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        out = append(out, m)
    }
    writeJSON(w, out)
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
        `PRAGMA journal_mode = WAL;`,
        `PRAGMA busy_timeout = 5000;`,

        `CREATE TABLE IF NOT EXISTS items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            text TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME
        );`,

        `CREATE TABLE IF NOT EXISTS matches (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            item_id INTEGER NOT NULL,
            matched_text TEXT,
            matched_url TEXT NOT NULL,
            source_site TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (item_id) REFERENCES items(id)
        );`,

        `CREATE UNIQUE INDEX IF NOT EXISTS ux_matches_dedupe
         ON matches(item_id, matched_url, source_site);`,
    }

    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return err
        }
    }
    return nil
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
