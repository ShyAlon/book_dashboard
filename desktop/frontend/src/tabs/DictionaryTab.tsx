import { DashboardData } from "../types";

type Props = { data: DashboardData };

export function DictionaryTab({ data }: Props) {
  return (
    <section className="panel-grid">
      <article className="panel">
        <h2>Character Dictionary</h2>
        {data.characterDictionary.length === 0 ? (
          <p>No characters extracted yet.</p>
        ) : (
          <ul className="list chapter-grid">
            {data.characterDictionary.map((c) => (
              <li key={c.name}>
                <strong>{c.name}</strong> <span className="muted">(mentions: {c.totalMentions})</span><br />
                <span className="muted">{c.description}</span><br />
                <span className="muted">First seen: Ch {c.firstSeenChapter} | Last seen: Ch {c.lastSeenChapter}</span>
                <ul className="list">
                  {c.chapters.slice(0, 3).map((ch) => (
                    <li key={`${c.name}-${ch.chapter}`}>
                      <strong>Ch {ch.chapter}:</strong> {ch.title}<br />
                      <span className="muted">{ch.summary}</span><br />
                      <span className="muted">Actions: {ch.actions.join(" | ")}</span>
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        )}
      </article>

      <article className="panel">
        <h2>Chapter Summaries</h2>
        <ul className="list chapter-grid">
          {data.chapterSummaries.map((c) => (
            <li key={`summary-${c.chapter}`}>
              <strong>Ch {c.chapter}:</strong> {c.title}<br />
              <span className="muted">{c.summary}</span><br />
              <span className="muted">Events: {c.events.join(" | ")}</span>
            </li>
          ))}
        </ul>
      </article>
    </section>
  );
}
