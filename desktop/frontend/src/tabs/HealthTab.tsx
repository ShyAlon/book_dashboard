import { DashboardData } from "../types";

type Props = {
  data: DashboardData;
  selectedIssue: number;
  setSelectedIssue: (n: number) => void;
};

export function HealthTab({ data, selectedIssue, setSelectedIssue }: Props) {
  const issues = data.healthIssues;
  return (
    <section className="panel-grid">
      <article className="panel">
        <h2>Contradictions</h2>
        {issues.length === 0 ? (
          <p>No contradictions found by heuristic extraction.</p>
        ) : (
          <ul className="list interactive">
            {issues.map((c, i) => (
              <li key={`${c.id}-${i}`} className={selectedIssue === i ? "selected" : ""} onClick={() => setSelectedIssue(i)}>
                <strong className={c.severity === "HIGH" ? "text-risk" : "text-warn"}>{c.severity}</strong> {c.description}
                <div className="log-detail">Ch {c.chapterA}: {c.contextA || "No context extracted."}</div>
                <div className="log-detail">Ch {c.chapterB}: {c.contextB || "No context extracted."}</div>
              </li>
            ))}
          </ul>
        )}
      </article>

      <article className="panel">
        <h2>Issue Drill-down</h2>
        {selectedIssue >= 0 && issues[selectedIssue] ? (
          <div className="issue-detail">
            <p><strong>Entity:</strong> {issues[selectedIssue].entity}</p>
            <p><strong>Chapter Pair:</strong> {issues[selectedIssue].chapterA} {"->"} {issues[selectedIssue].chapterB}</p>
            <p><strong>Dictionary Ref:</strong> {issues[selectedIssue].dictionaryRef}</p>
            <p><strong>Context A:</strong> {issues[selectedIssue].contextA}</p>
            <p><strong>Context B:</strong> {issues[selectedIssue].contextB}</p>
          </div>
        ) : (
          <p>Select a contradiction from the list to inspect details.</p>
        )}
        <h2>Summary</h2>
        <p>Health focuses on contradiction and consistency checks.</p>
        <p>Use the <strong>AI Detection</strong> tab for AI-likelihood signals and flags.</p>
      </article>
    </section>
  );
}
