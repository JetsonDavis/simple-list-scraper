import { useEffect, useState } from "react";

type Item = { id: number; text: string };
type Match = { id: number; item: string; url: string; site: string; created: string };

export default function App() {
  const [tab, setTab] = useState<"items" | "matches">("items");
  const [items, setItems] = useState<Item[]>([]);
  const [matches, setMatches] = useState<Match[]>([]);
  const [text, setText] = useState("");

  async function loadItems() {
    const res = await fetch("/api/items");
    setItems(await res.json());
  }
  async function loadMatches() {
    const res = await fetch("/api/matches");
    setMatches(await res.json());
  }

  useEffect(() => { loadItems(); }, []);
  useEffect(() => { if (tab === "matches") loadMatches(); }, [tab]);

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
    await fetch(`/api/items/${id}`, { method: "DELETE" });
    await loadItems();
  }

  return (
    <div className="container">
      <h2>Simple List</h2>

      <div className="tabs">
        <button className={"tab " + (tab === "items" ? "active" : "")} onClick={() => setTab("items")}>
          Items
        </button>
        <button className={"tab " + (tab === "matches" ? "active" : "")} onClick={() => setTab("matches")}>
          Matches
        </button>
      </div>

      {tab === "items" && (
        <>
          <div className="row" style={{ marginTop: 12 }}>
            <input
              type="text"
              value={text}
              placeholder="Type an item…"
              onChange={(e) => setText(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && add()}
            />
            <button onClick={add}>Add</button>
          </div>

          <div className="list">
            {items.map((i) => (
              <div className="item" key={i.id}>
                <input
                  type="text"
                  defaultValue={i.text}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                  }}
                  onBlur={(e) => update(i.id, e.target.value)}
                  title="Edit and click away (or press Enter) to save"
                />
                <button className="small" onClick={() => remove(i.id)}>Delete</button>
              </div>
            ))}
          </div>

          <div className="hint">
            Worker runs every 6 hours by default. Override with <span className="badge">CHECK_INTERVAL_HOURS</span>.
          </div>
        </>
      )}

      {tab === "matches" && (
        <>
          <div className="hint">
            Matches are deduped on (item_id, matched_url, source_site). New inserts can trigger Twilio SMS if configured.
          </div>
          <table className="table">
            <thead>
              <tr>
                <th>Item</th>
                <th>Site</th>
                <th>URL</th>
                <th>When</th>
              </tr>
            </thead>
            <tbody>
              {matches.map((m) => (
                <tr key={m.id}>
                  <td>{m.item}</td>
                  <td>{m.site}</td>
                  <td><a href={m.url} target="_blank" rel="noreferrer">{m.url}</a></td>
                  <td>{m.created}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </div>
  );
}
