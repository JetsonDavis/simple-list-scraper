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
  const [polling, setPolling] = useState(false);

  async function loadItems() {
    const res = await fetch("/api/items");
    setItems(await res.json());
  }
  async function loadUrls() {
    const res = await fetch("/api/urls");
    setUrls(await res.json());
  }
  async function loadMatches() {
    const res = await fetch("/api/matches");
    setMatches(await res.json());
  }

  useEffect(() => { loadItems(); loadUrls(); }, []);
  useEffect(() => {
    localStorage.setItem("selectedTab", tab);
    if (tab === "matches") loadMatches();
  }, [tab]);

  // Poll matches while worker is running
  useEffect(() => {
    if (!polling) return;

    const interval = setInterval(() => {
      loadMatches();
    }, 2000); // Poll every 2 seconds

    return () => clearInterval(interval);
  }, [polling]);

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
      alert(`Error deleting item: ${err}`);
    }
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
    setTriggering(true);
    setPolling(true); // Start polling for updates
    try {
      const res = await fetch("/api/trigger-worker", { method: "POST" });
      const data = await res.json();
      
      // Poll worker status until it completes
      const checkWorkerStatus = async () => {
        try {
          const statusRes = await fetch("/api/trigger-worker", { method: "POST" });
          const statusData = await statusRes.json();
          
          // If worker is already running, keep checking
          if (statusData.status === "already_running") {
            setTimeout(checkWorkerStatus, 2000); // Check again in 2 seconds
          } else {
            // Worker has finished
            setPolling(false);
            setTriggering(false);
            await loadMatches(); // Final load
            alert("Worker completed");
          }
        } catch (err) {
          // If there's an error, assume worker finished
          setPolling(false);
          setTriggering(false);
          await loadMatches();
        }
      };
      
      // Start checking after 2 seconds
      setTimeout(checkWorkerStatus, 2000);
    } catch (err) {
      alert("Failed to trigger worker");
      setPolling(false);
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
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
