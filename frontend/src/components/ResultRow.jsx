// One row that is both the voting button and the live result bar.
export default function ResultRow({ text, count, total, lead, mine, interactive, busy, onPick }) {
  const pct = total ? (count / total) * 100 : 0;
  const className = `row${interactive ? " pick" : ""}${lead ? " lead" : ""}${mine ? " mine" : ""}`;
  const inner = (
    <>
      <span className="fill" style={{ width: `${pct}%` }} aria-hidden="true" />
      <span className="row-text">
        {text}
        {mine && <em className="you">Your vote</em>}
      </span>
      <span className="row-num">
        <strong>{count}</strong>
        <span>{Math.round(pct)}%</span>
      </span>
    </>
  );

  if (interactive) {
    return (
      <button type="button" className={className} onClick={onPick} disabled={busy}>
        {inner}
      </button>
    );
  }
  return <div className={className}>{inner}</div>;
}
