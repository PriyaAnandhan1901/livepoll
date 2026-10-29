import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth.jsx";
import ResultRow from "../components/ResultRow.jsx";

const SAMPLE = ["Tea", "Coffee", "Neither"];

function SamplePoll() {
  const [counts, setCounts] = useState([12, 15, 3]);

  useEffect(() => {
    const t = setInterval(() => {
      setCounts((prev) => {
        const next = [...prev];
        next[Math.floor(Math.random() * next.length)] += 1;
        return next;
      });
    }, 1400);
    return () => clearInterval(t);
  }, []);

  const total = counts.reduce((a, b) => a + b, 0);
  const max = Math.max(...counts);

  return (
    <section className="sample" aria-label="Sample poll with simulated votes">
      <h2>Tea or coffee?</h2>
      <div className="rows">
        {SAMPLE.map((text, i) => (
          <ResultRow key={text} text={text} count={counts[i]} total={total} lead={counts[i] === max} />
        ))}
      </div>
      <p className="muted small-text">Sample poll with simulated votes. Real polls update the same way.</p>
    </section>
  );
}

export default function Home() {
  const { user } = useAuth();
  return (
    <main className="page hero">
      <div>
        <h1>Ask a question. Watch the answers arrive.</h1>
        <p className="lead-text">
          Create a poll, share the link, and everyone sees the results change the moment a vote comes in. No refresh, no waiting.
        </p>
        <div className="actions">
          <Link className="btn" to={user ? "/create" : "/signup"}>
            {user ? "Create a poll" : "Create your first poll"}
          </Link>
          {!user && <Link className="btn ghost" to="/login">Log in</Link>}
        </div>
      </div>
      <SamplePoll />
    </main>
  );
}
