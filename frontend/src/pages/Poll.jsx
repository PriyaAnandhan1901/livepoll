import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { getVoted, getVoterId, request, setVoted as rememberVote, streamUrl } from "../api.js";
import ResultRow from "../components/ResultRow.jsx";

function remaining(ms) {
  const s = Math.max(0, Math.floor(ms / 1000));
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m ${s % 60}s`;
}

export default function Poll() {
  const { id } = useParams();
  const [poll, setPoll] = useState(null);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("connecting");
  const [voted, setVoted] = useState(() => getVoted(id));
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const [copied, setCopied] = useState(false);
  const [now, setNow] = useState(Date.now());

  // Load the poll, then subscribe to live updates.
  useEffect(() => {
    let es;
    let cancelled = false;
    setPoll(null);
    setError("");
    setStatus("connecting");
    setVoted(getVoted(id));

    request(`/polls/${id}`)
      .then((p) => {
        if (cancelled) return;
        setPoll(p);
        document.title = `${p.question} | Livepoll`;
        es = new EventSource(streamUrl(id));
        es.onopen = () => setStatus("live");
        es.onerror = () => setStatus("reconnecting");
        es.onmessage = (e) => {
          try {
            const r = JSON.parse(e.data);
            setPoll((prev) => (prev ? { ...prev, counts: r.counts, total: r.total, open: r.open } : prev));
          } catch {
            /* ignore malformed event */
          }
        };
      })
      .catch((err) => {
        if (!cancelled) setError(err.status === 404 ? "This poll doesn't exist or was deleted." : err.message);
      });

    return () => {
      cancelled = true;
      if (es) es.close();
    };
  }, [id]);

  // Tick once a second only when the poll has an expiry.
  useEffect(() => {
    if (!poll?.expiresAt) return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [poll?.expiresAt]);

  if (error) {
    return (
      <main className="page narrow">
        <h1>Poll unavailable</h1>
        <p className="error">{error}</p>
        <Link className="btn" to="/">Go to home</Link>
      </main>
    );
  }
  if (!poll) {
    return (
      <main className="page narrow">
        <p className="muted">Loading poll</p>
      </main>
    );
  }

  const expiresMs = poll.expiresAt ? Date.parse(poll.expiresAt) - now : null;
  const open = poll.open && (expiresMs === null || expiresMs > 0);
  const canVote = open && voted === null;
  const max = Math.max(...poll.counts);

  async function vote(i) {
    if (busy || !canVote) return;
    setBusy(true);
    setNotice("");
    try {
      const r = await request(`/polls/${id}/vote`, {
        method: "POST",
        body: { optionIndex: i, voterId: getVoterId() },
      });
      setPoll((p) => ({ ...p, counts: r.counts, total: r.total, open: r.open }));
      setVoted(i);
      rememberVote(id, i);
    } catch (err) {
      if (err.status === 409 && /already/.test(err.message)) {
        setVoted(-1);
        rememberVote(id, -1);
        setNotice("You've already voted in this poll.");
      } else {
        setNotice(err.message);
      }
    } finally {
      setBusy(false);
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      setNotice("Couldn't copy automatically. Copy the address from your browser's address bar.");
    }
  }

  const shareText = encodeURIComponent(`Vote on: ${poll.question} ${window.location.href}`);

  return (
    <main className="page narrow">
      <div className="meta">
        <span className={`pill ${open && status === "live" ? "on" : ""}`}>
          <span className="dot" aria-hidden="true" />
          {!open ? "Closed" : status === "live" ? "Live" : status === "reconnecting" ? "Reconnecting" : "Connecting"}
        </span>
        <span className="muted">{poll.total} {poll.total === 1 ? "vote" : "votes"}</span>
        {open && expiresMs !== null && <span className="muted">Closes in {remaining(expiresMs)}</span>}
      </div>

      <h1 className="question">{poll.question}</h1>

      <div className="rows" aria-live="polite">
        {poll.options.map((text, i) => (
          <ResultRow
            key={i}
            text={text}
            count={poll.counts[i] ?? 0}
            total={poll.total}
            lead={max > 0 && poll.counts[i] === max}
            mine={voted === i}
            interactive={canVote}
            busy={busy}
            onPick={() => vote(i)}
          />
        ))}
      </div>

      <p className="muted hint">
        {!open
          ? "Voting has closed. These are the final results."
          : voted === null
          ? "Pick an option to cast your vote."
          : "Your vote is counted. Results update as others vote."}
      </p>
      {notice && <p className="error" role="alert">{notice}</p>}

      <div className="share">
        <button className="btn ghost small" onClick={copy}>{copied ? "Link copied" : "Copy link"}</button>
        <a className="btn ghost small" href={`https://wa.me/?text=${shareText}`} target="_blank" rel="noreferrer">
          Share on WhatsApp
        </a>
      </div>
    </main>
  );
}
