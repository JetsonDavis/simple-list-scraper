import { useEffect, useState } from "react";

type Item = { id: number; text: string };
type URL = { id: number; url: string; display_name?: string; config?: string };
type Match = { id: number; item: string; url: string; site: string; torrent_text?: string; created: string };

export default function App() {
  const [tab, setTab] = useState<"items" | "urls" | "matches">(() => {
    const savedTab = localStorage.getItem("selectedTab");
    return (savedTab === "items" || savedTab === "urls" || savedTab === "matches") ? savedTab : "items";
  });
  const [items, setItems] = useState<Item[]>([]);
  const [urls, setUrls] = useState<URL[]>([]);
  const [matches, setMatches] = useState<Match[]>([]);
  const [text, setText] = useState("");
  const [urlText, setUrlText] = useState("");
  const [triggering, setTriggering] = useState(false);
  const [ws, setWs] = useState<WebSocket | null>(null);

  async function loadItems() {
    try {
      const res = await fetch("/api/items");
      if (!res.ok) {
        console.error("Failed to load items:", res.status);
        return;
      }
      setItems(await res.json());
    } catch (err) {
      console.error("Error loading items:", err);
    }
  }
  async function loadUrls() {
    try {
      const res = await fetch("/api/urls");
      if (!res.ok) {
        console.error("Failed to load URLs:", res.status);
        return;
      }
      setUrls(await res.json());
    } catch (err) {
      console.error("Error loading URLs:", err);
    }
  }
  async function loadMatches() {
    try {
      const res = await fetch("/api/matches");
      if (!res.ok) {
        console.error("Failed to load matches:", res.status);
        return;
      }
      setMatches(await res.json());
    } catch (err) {
      console.error("Error loading matches:", err);
    }
  }

  useEffect(() => { loadItems(); loadUrls(); }, []);
  useEffect(() => {
    localStorage.setItem("selectedTab", tab);
    if (tab === "matches") loadMatches();
  }, [tab]);

  // WebSocket connection for real-time updates
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/ws`;
    const websocket = new WebSocket(wsUrl);

    websocket.onopen = () => {
      console.log('WebSocket connected');
    };

    websocket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.type === 'worker_status') {
        console.log('Worker status:', data.status, data.message);
        if (data.status === 'running') {
          setTriggering(true);
        } else if (data.status === 'completed') {
          setTriggering(false);
          loadMatches(); // Final load when worker completes
        }
      } else if (data.type === 'new_match') {
        console.log('New match:', data.match);
        // Add new match to the list
        setMatches(prev => [data.match, ...prev]);
      }
    };

    websocket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    websocket.onclose = () => {
      console.log('WebSocket disconnected');
    };

    setWs(websocket);

    return () => {
      websocket.close();
    };
  }, []);

  async function add() {
    const v = text.trim();
    if (!v) return;
    await fetch("/api/items", {
      method: "POST",
      body: new URLSearchParams({ text: v }),
    });
    setText("");
    await loadItems();
  }

  async function update(id: number, newText: string) {
    const v = newText.trim();
    if (!v) return;
    await fetch(`/api/items/${id}`, {
      method: "PUT",
      body: new URLSearchParams({ text: v }),
    });
    await loadItems();
  }

  async function remove(id: number) {
    if (!confirm("Delete this item?")) return;
    try {
      const res = await fetch(`/api/items/${id}`, { method: "DELETE" });
      if (!res.ok) {
        const text = await res.text();
        console.error("Delete failed:", res.status, text);
        alert(`Failed to delete item: ${text}`);
        return;
      }
      await loadItems();
    } catch (err) {
      console.error("Delete error:", err);
      alert("Failed to delete item");
      return;
    }
  }

  async function removeMatch(id: number) {
    if (!confirm("Delete this match?")) return;
    try {
      const res = await fetch(`/api/matches/${id}`, { method: "DELETE" });
      if (!res.ok) {
        const text = await res.text();
        console.error("Delete match failed:", res.status, text);
        alert(`Failed to delete match: ${text}`);
        return;
      }
    } catch (err) {
      console.error("Delete match error:", err);
      alert("Failed to delete match");
      return;
    }
    await loadMatches();
  }

  async function addUrl() {
    const v = urlText.trim();
    if (!v) return;
    await fetch("/api/urls", {
      method: "POST",
      body: new URLSearchParams({ url: v }),
    });
    setUrlText("");
    await loadUrls();
  }

  async function updateUrl(id: number, newUrl: string, newDisplayName?: string) {
    const v = newUrl.trim();
    if (!v && !newDisplayName) return;
    const params = new URLSearchParams();
    if (v) params.append("url", v);
    if (newDisplayName !== undefined) params.append("display_name", newDisplayName.trim());
    await fetch(`/api/urls/${id}`, {
      method: "PUT",
      body: params,
    });
    await loadUrls();
  }

  async function removeUrl(id: number) {
    if (!confirm("Delete this URL?")) return;
    await fetch(`/api/urls/${id}`, { method: "DELETE" });
    await loadUrls();
  }

  async function triggerWorker() {
    try {
      const res = await fetch("/api/trigger-worker", { method: "POST" });
      const data = await res.json();
      
      if (data.status === "already_running") {
        alert("Worker is already running");
      } else if (data.status === "triggered") {
        // Set spinner immediately when worker starts
        setTriggering(true);
        // WebSocket will handle updates and turn off spinner when completed
      }
    } catch (err) {
      alert("Failed to trigger worker");
      setTriggering(false);
    }
  }

  return (
    <div className="container">
      <div className="header">
        <h2>Torrent Seeker</h2>
        <button onClick={triggerWorker} disabled={triggering} className="run-button">
          {triggering && <span className="spinner"></span>}
          {triggering ? "Running..." : "Run"}
        </button>
      </div>

      <div className="tabs">
        <button className={"tab " + (tab === "items" ? "active" : "")} onClick={() => setTab("items")}>
          Items
        </button>
        <button className={"tab " + (tab === "urls" ? "active" : "")} onClick={() => setTab("urls")}>
          URLs
        </button>
        <button className={"tab " + (tab === "matches" ? "active" : "")} onClick={() => setTab("matches")}>
          Matches
        </button>
      </div>

      {tab === "items" && (
        <div className="tab-content">
          <div className="row">
            <input
              type="text"
              value={text}
              placeholder="Type an item…"
              onChange={(e) => setText(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && add()}
            />
            <button onClick={add}>Add</button>
          </div>

          <table className="table">
            <thead>
              <tr>
                <th>Item</th>
                <th style={{ width: '100px', textAlign: 'center' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((i) => (
                <tr key={i.id}>
                  <td>
                    <input
                      type="text"
                      defaultValue={i.text}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                      }}
                      onBlur={(e) => update(i.id, e.target.value)}
                      title="Edit and click away (or press Enter) to save"
                      style={{ width: '100%', border: 'none', background: 'transparent', padding: '4px' }}
                    />
                  </td>
                  <td style={{ textAlign: 'center' }}>
                    <button className="small" onClick={() => remove(i.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="hint">
            Worker runs every 6 hours by default. Override with <span className="badge">CHECK_INTERVAL_HOURS</span>.
          </div>
        </div>
      )}

      {tab === "urls" && (
        <div className="tab-content">
          <div className="row">
            <input
              type="text"
              value={urlText}
              placeholder="Enter a URL to scrape…"
              onChange={(e) => setUrlText(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && addUrl()}
            />
            <button onClick={addUrl}>Add</button>
          </div>

          <table className="table">
            <thead>
              <tr>
                <th>Display Name</th>
                <th>URL</th>
                <th style={{ width: '100px', textAlign: 'center' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {urls.map((u) => (
                <tr key={u.id}>
                  <td style={{ width: '200px' }}>
                    <input
                      type="text"
                      defaultValue={u.display_name || ''}
                      placeholder="Display name..."
                      onKeyDown={(e) => {
                        if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                      }}
                      onBlur={(e) => updateUrl(u.id, u.url, e.target.value)}
                      title="Edit and click away (or press Enter) to save"
                      style={{ width: '100%', border: 'none', background: 'transparent', padding: '4px' }}
                    />
                  </td>
                  <td>
                    <input
                      type="text"
                      defaultValue={u.url}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                      }}
                      onBlur={(e) => updateUrl(u.id, e.target.value, u.display_name)}
                      title="Edit and click away (or press Enter) to save"
                      style={{ width: '100%', border: 'none', background: 'transparent', padding: '4px' }}
                    />
                  </td>
                  <td style={{ textAlign: 'center' }}>
                    <button className="small" onClick={() => removeUrl(u.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="hint">
            These URLs will be scraped for each item in your list.
          </div>
        </div>
      )}

      {tab === "matches" && (
        <div className="tab-content">
          <div className="hint">
            Matches are deduped on (item_id, matched_url, source_site). New inserts can trigger Twilio SMS if configured.
          </div>
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: '20%' }}>Item</th>
                <th style={{ width: '10%' }}>Site</th>
                <th style={{ width: '30%' }}>Torrent Text</th>
                <th>URL</th>
                <th style={{ width: '180px' }}>When</th>
                <th style={{ width: '80px', textAlign: 'center' }}>Action</th>
              </tr>
            </thead>
            <tbody>
              {matches.map((m) => (
                <tr key={m.id}>
                  <td>{m.item}</td>
                  <td>{m.site}</td>
                  <td>{m.torrent_text || ''}</td>
                  <td><a href={m.url} target="_blank" rel="noreferrer">{m.url}</a></td>
                  <td>{m.created}</td>
                  <td style={{ textAlign: 'center' }}>
                    <button 
                      className="small" 
                      onClick={() => removeMatch(m.id)}
                      style={{ 
                        background: 'transparent', 
                        border: 'none', 
                        cursor: 'pointer',
                        padding: '4px'
                      }}
                      title="Delete match"
                    >
                      <svg 
                        width="18" 
                        height="18" 
                        viewBox="0 0 24 24" 
                        fill="none" 
                        stroke="#D13438" 
                        strokeWidth="2" 
                        strokeLinecap="round" 
                        strokeLinejoin="round"
                      >
                        <polyline points="3 6 5 6 21 6"></polyline>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                        <line x1="10" y1="11" x2="10" y2="17"></line>
                        <line x1="14" y1="11" x2="14" y2="17"></line>
                      </svg>
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
