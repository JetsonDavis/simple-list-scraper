import { useEffect, useState } from "react";

type Item = { id: number; text: string };
type URL = { id: number; url: string; display_name?: string; config?: string };
type Match = { id: number; item: string; url: string; site: string; torrent_text?: string; magnet_link?: string; created: string };
type Log = { id: number; timestamp: string; description: string; success: boolean };
type LogsResponse = { logs: Log[]; page: number; page_size: number; total: number; total_pages: number };

export default function App() {
  const [tab, setTab] = useState<"items" | "urls" | "matches" | "logs">(() => {
    const savedTab = localStorage.getItem("selectedTab");
    return (savedTab === "items" || savedTab === "urls" || savedTab === "matches" || savedTab === "logs") ? savedTab : "items";
  });
  const [items, setItems] = useState<Item[]>([]);
  const [urls, setUrls] = useState<URL[]>([]);
  const [matches, setMatches] = useState<Match[]>([]);
  const [logs, setLogs] = useState<Log[]>([]);
  const [logsPage, setLogsPage] = useState(1);
  const [logsTotalPages, setLogsTotalPages] = useState(0);
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

  async function loadLogs(page: number = 1) {
    try {
      const res = await fetch(`/api/logs?page=${page}`);
      if (!res.ok) {
        console.error("Failed to load logs:", res.status);
        return;
      }
      const data: LogsResponse = await res.json();
      setLogs(data.logs);
      setLogsPage(data.page);
      setLogsTotalPages(data.total_pages);
    } catch (err) {
      console.error("Error loading logs:", err);
    }
  }

  useEffect(() => { loadItems(); loadUrls(); }, []);
  useEffect(() => {
    localStorage.setItem("selectedTab", tab);
    if (tab === "matches") {
      loadMatches();
      // Check worker status when opening matches tab to ensure spinner is correct
      fetch("/api/worker-status")
        .then(res => res.json())
        .then(data => {
          if (data.running === false) {
            setTriggering(false);
          }
        })
        .catch(err => console.error("Failed to check worker status:", err));
    } else if (tab === "logs") {
      loadLogs(1);
    }
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
      } else if (data.type === 'new_log') {
        console.log('New log:', data.log);
        // Add new log to the list if on logs tab
        setLogs(prev => [data.log, ...prev]);
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
    try {
      console.log("Adding item:", v);
      const res = await fetch("/api/items", {
        method: "POST",
        body: new URLSearchParams({ text: v }),
      });
      if (!res.ok) {
        if (res.status === 409) {
          // Conflict - item already exists
          const data = await res.json();
          alert(data.error || "Item already exists");
          return;
        }
        console.error("Failed to add item:", res.status);
        alert(`Failed to add item: ${res.status}`);
        return;
      }
      console.log("Item added successfully, clearing input and reloading...");
      setText("");
      await loadItems();
      console.log("Items reloaded, new count:", items.length);
    } catch (err) {
      console.error("Error adding item:", err);
      alert("Failed to add item");
    }
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
    console.log("removeMatch called with ID:", id);
    try {
      const url = `/api/matches/${id}`;
      console.log("Sending DELETE request to:", url);
      const res = await fetch(url, { method: "DELETE" });
      if (!res.ok) {
        const text = await res.text();
        console.error("Delete match failed:", res.status, text);
        alert(`Failed to delete match: ${text}`);
        return;
      }
      console.log("Delete successful, reloading matches...");
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

  async function clearLogs() {
    if (!confirm("Delete all logs? This cannot be undone.")) return;
    try {
      const res = await fetch("/api/logs", { method: "DELETE" });
      if (!res.ok) {
        alert("Failed to clear logs");
        return;
      }
      setLogs([]);
      setLogsPage(1);
      setLogsTotalPages(0);
    } catch (err) {
      console.error("Error clearing logs:", err);
      alert("Failed to clear logs");
    }
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
        <button className={"tab " + (tab === "matches" ? "active" : "")} onClick={() => setTab("matches")}>
          Matches
        </button>
        <button className={"tab " + (tab === "urls" ? "active" : "")} onClick={() => setTab("urls")}>
          Sites
        </button>
        <button className={"tab " + (tab === "logs" ? "active" : "")} onClick={() => setTab("logs")}>
          Logs
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

          <div className="hint">
            These URLs will be scraped for each item in your list.
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
        </div>
      )}

      {tab === "matches" && (
        <div className="tab-content">
          <table className="table" style={{ width: '100%' }}>
            <thead>
              <tr>
                <th style={{ width: '15%' }}>ITEM</th>
                <th style={{ width: '10%' }}>SITE</th>
                <th style={{ width: '55%' }}>TORRENT TEXT</th>
                <th style={{ width: '8%', textAlign: 'center' }}>MAGNET</th>
                <th style={{ width: '12%' }}>WHEN</th>
                <th style={{ width: '8%', textAlign: 'center' }}>ACTION</th>
              </tr>
            </thead>
            <tbody>
              {matches.map((m) => (
                <tr key={m.id}>
                  <td>{m.item}</td>
                  <td>{m.site}</td>
                  <td>
                    <a href={m.url} target="_blank" rel="noreferrer" style={{ color: '#0078D4', textDecoration: 'none' }}>
                      {m.torrent_text || m.url}
                    </a>
                  </td>
                  <td style={{ textAlign: 'center' }}>
                    {m.magnet_link ? (
                      <a href={m.magnet_link} title="Open magnet link">
                        <svg
                          width="18"
                          height="18"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="#0078D4"
                          strokeWidth="2"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        >
                          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                          <polyline points="7 10 12 15 17 10"></polyline>
                          <line x1="12" y1="15" x2="12" y2="3"></line>
                        </svg>
                      </a>
                    ) : (
                      <span style={{ color: '#999' }}>-</span>
                    )}
                  </td>
                  <td style={{ whiteSpace: 'nowrap' }}>{m.created}</td>
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

      {tab === "logs" && (
        <div className="tab-content">
          <div className="clear_logs">
            <button onClick={() => clearLogs()}>Clear Logs</button>
          </div>
          <table className="table" style={{ width: '100%' }}>
            <thead>
              <tr>
                <th style={{ width: '180px' }}>TIMESTAMP</th>
                <th style={{ width: 'auto' }}>DESCRIPTION</th>
                <th style={{ width: '100px', textAlign: 'center' }}>STATUS</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id}>
                  <td style={{ whiteSpace: 'nowrap', fontSize: '13px' }}>
                    {new Date(log.timestamp).toLocaleString()}
                  </td>
                  <td>{log.description}</td>
                  <td style={{ textAlign: 'center' }}>
                    <span style={{
                      padding: '4px 8px',
                      borderRadius: '4px',
                      fontSize: '12px',
                      fontWeight: 'bold',
                      backgroundColor: log.success ? '#d4edda' : '#f8d7da',
                      color: log.success ? '#155724' : '#721c24'
                    }}>
                      {log.success ? 'SUCCESS' : 'FAILED'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {logsTotalPages > 1 && (
            <div style={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              gap: '10px',
              marginTop: '20px'
            }}>
              <button
                onClick={() => loadLogs(logsPage - 1)}
                disabled={logsPage === 1}
                style={{ padding: '8px 16px' }}
              >
                Previous
              </button>
              <span>Page {logsPage} of {logsTotalPages}</span>
              <button
                onClick={() => loadLogs(logsPage + 1)}
                disabled={logsPage === logsTotalPages}
                style={{ padding: '8px 16px' }}
              >
                Next
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
