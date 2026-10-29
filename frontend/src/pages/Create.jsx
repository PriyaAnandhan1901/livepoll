import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { request } from "../api.js";
import { useAuth } from "../auth.jsx";

const EXPIRY = [
  { label: "No expiry", minutes: 0 },
  { label: "10 minutes", minutes: 10 },
  { label: "1 hour", minutes: 60 },
  { label: "1 day", minutes: 1440 },
  { label: "7 days", minutes: 10080 },
];

export default function Create() {
  const navigate = useNavigate();
  const { logout } = useAuth();
  const [question, setQuestion] = useState("");
  const [options, setOptions] = useState(["", ""]);
  const [expires, setExpires] = useState(0);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const setOption = (i, v) => setOptions((o) => o.map((x, j) => (j === i ? v : x)));

  async function submit(e) {
    e.preventDefault();
    setError("");
    const q = question.trim();
    const opts = options.map((o) => o.trim()).filter(Boolean);
    if (q.length < 3) return setError("Write a question of at least 3 characters.");
    if (opts.length < 2) return setError("Add at least two options.");
    if (new Set(opts.map((o) => o.toLowerCase())).size !== opts.length) return setError("Options must be different from each other.");
    setBusy(true);
    try {
      const poll = await request("/polls", {
        method: "POST",
        auth: true,
        body: { question: q, options: opts, expiresInMinutes: expires },
      });
      navigate(`/p/${poll.id}`);
    } catch (err) {
      if (err.status === 401) {
        logout();
        navigate("/login", { state: { from: "/create" } });
        return;
      }
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="page narrow">
      <h1>New poll</h1>
      <form onSubmit={submit} className="form" noValidate>
        <label>
          Question
          <input value={question} onChange={(e) => setQuestion(e.target.value)} maxLength={200} placeholder="Where should we eat on Friday?" />
        </label>

        <fieldset>
          <legend>Options</legend>
          {options.map((o, i) => (
            <div className="option-input" key={i}>
              <input
                value={o}
                onChange={(e) => setOption(i, e.target.value)}
                maxLength={100}
                aria-label={`Option ${i + 1}`}
                placeholder={`Option ${i + 1}`}
              />
              {options.length > 2 && (
                <button type="button" className="link-btn" onClick={() => setOptions((x) => x.filter((_, j) => j !== i))}>
                  Remove
                </button>
              )}
            </div>
          ))}
          {options.length < 10 && (
            <button type="button" className="btn ghost small" onClick={() => setOptions((o) => [...o, ""])}>
              Add option
            </button>
          )}
        </fieldset>

        <label>
          Close the poll after
          <select value={expires} onChange={(e) => setExpires(Number(e.target.value))}>
            {EXPIRY.map((x) => (
              <option key={x.minutes} value={x.minutes}>{x.label}</option>
            ))}
          </select>
        </label>

        {error && <p className="error" role="alert">{error}</p>}
        <button className="btn" disabled={busy}>{busy ? "Creating" : "Create poll"}</button>
      </form>
    </main>
  );
}
