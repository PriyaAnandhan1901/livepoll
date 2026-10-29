import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { request } from "../api.js";
import { useAuth } from "../auth.jsx";

export default function Dashboard() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [polls, setPolls] = useState(null);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState("");

  function handle(err) {
    if (err.status === 401) {
      logout();
      navigate("/login", { state: { from: "/dashboard" } });
    } else {
      setError(err.message);
    }
  }

  useEffect(() => {
    request("/my/polls", { auth: true }).then(setPolls).catch(handle);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const link = (id) => `${window.location.origin}/p/${id}`;

  async function copy(id) {
    try {
      await navigator.clipboard.writeText(link(id));
      setCopied(id);
      setTimeout(() => setCopied(""), 1800);
    } catch {
      setError("Couldn't copy the link. Select it from the poll page instead.");
    }
  }

  async function close(id) {
    try {
      await request(`/polls/${id}/close`, { method: "POST", auth: true });
      setPolls((p) => p.map((x) => (x.id === id ? { ...x, open: false, closed: true } : x)));
    } catch (err) {
      handle(err);
    }
  }

  async function remove(id) {
    if (!window.confirm("Delete this poll and all its votes? This can't be undone.")) return;
    try {
      await request(`/polls/${id}`, { method: "DELETE", auth: true });
      setPolls((p) => p.filter((x) => x.id !== id));
    } catch (err) {
      handle(err);
    }
  }

  return (
    <main className="page">
      <div className="page-head">
        <h1>{user?.name ? `${user.name}'s polls` : "My polls"}</h1>
        <Link className="btn" to="/create">New poll</Link>
      </div>
      {error && <p className="error" role="alert">{error}</p>}
      {polls === null && !error && <p className="muted">Loading your polls</p>}
      {polls && polls.length === 0 && (
        <div className="empty">
          <h2>No polls yet</h2>
          <p className="muted">Create one, share the link, and watch the votes come in.</p>
          <Link className="btn" to="/create">Create a poll</Link>
        </div>
      )}
      <ul className="poll-list">
        {polls?.map((p) => (
          <li key={p.id} className="poll-item">
            <div>
              <Link to={`/p/${p.id}`} className="poll-title">{p.question}</Link>
              <p className="muted small-text">
                {p.total} {p.total === 1 ? "vote" : "votes"}, {p.open ? "open" : "closed"}
              </p>
            </div>
            <div className="item-actions">
              <button className="btn ghost small" onClick={() => copy(p.id)}>
                {copied === p.id ? "Link copied" : "Copy link"}
              </button>
              {p.open && <button className="btn ghost small" onClick={() => close(p.id)}>Close poll</button>}
              <button className="btn ghost small danger" onClick={() => remove(p.id)}>Delete</button>
            </div>
          </li>
        ))}
      </ul>
    </main>
  );
}
