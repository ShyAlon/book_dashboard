import { DashboardData } from "../types";

type Props = { data: DashboardData };

export function LanguageTab({ data }: Props) {
  const spellingProvider = data.language.spellingProvider || "heuristic";
  const safetyProvider = data.language.safetyProvider || "heuristic";

  return (
    <section className="panel-grid">
      <article className="panel">
        <h2>Language Quality</h2>
        {data.language.heuristicFallback ? <p className="text-risk"><strong>Fallback Warning:</strong> Heuristic mode is active for part of language analysis.</p> : null}
        <ul className="list">
          <li><strong>Spelling Score:</strong> {data.language.spellingScore}/100</li>
          <li><strong>Grammar Score:</strong> {data.language.grammarScore}/100</li>
          <li><strong>Readability Score:</strong> {data.language.readabilityScore}/100</li>
          <li><strong>Spelling & Grammar Provider:</strong> {spellingProvider}</li>
          <li><strong>Age Category:</strong> {data.language.ageCategory}</li>
        </ul>
      </article>
      <article className="panel">
        <h2>Content Safety</h2>
        <ul className="list">
          <li><strong>Safety Provider:</strong> {safetyProvider}</li>
          <li><strong>Profanity Score:</strong> {data.language.profanityScore}/100 ({data.language.profanityInstances} instances)</li>
          <li><strong>Explicit Score:</strong> {data.language.explicitScore}/100 ({data.language.explicitInstances} instances)</li>
          <li><strong>Violence Score:</strong> {data.language.violenceScore}/100</li>
        </ul>
      </article>
      <article className="panel panel-wide">
        <h2>Additional Diagnostics</h2>
        <ul className="list">
          {data.language.notes.map((n, i) => (
            <li key={`${n}-${i}`}>{n}</li>
          ))}
        </ul>
      </article>
    </section>
  );
}
