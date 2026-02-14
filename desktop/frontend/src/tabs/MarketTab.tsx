import { Radar, RadarChart, PolarGrid, PolarAngleAxis, ResponsiveContainer } from "recharts";
import { DashboardData } from "../types";

type Props = { data: DashboardData };

export function MarketTab({ data }: Props) {
  const radarData = data.genreScores.map((x) => ({ genre: x.genre, score: Math.round(x.score * 100) }));

  return (
    <section className="panel-grid">
      <article className="panel">
        <h2>Genre Radar</h2>
        <div className="chart-wrap">
          <ResponsiveContainer width="100%" height={280}>
            <RadarChart data={radarData}>
              <PolarGrid stroke="#3f3f46" />
              <PolarAngleAxis dataKey="genre" tick={{ fill: "#d4d4d8", fontSize: 12 }} />
              <Radar dataKey="score" stroke="#10b981" fill="#10b981" fillOpacity={0.38} />
            </RadarChart>
          </ResponsiveContainer>
        </div>
      </article>
      <article className="panel">
        <h2>Comp Titles</h2>
        <ul className="list">
          {data.compTitles.map((t) => (
            <li key={t.title}>
              <strong>{t.title}</strong> <span className="muted">({t.tier})</span>
            </li>
          ))}
        </ul>
      </article>
      <article className="panel panel-wide">
        <h2>Genre by Chapter</h2>
        <ul className="list chapter-grid">
          {data.chapterMetrics.map((c) => (
            <li key={`${c.index}-${c.title}`}>
              <strong>Ch {c.index}:</strong> {c.title}<br />
              <span className="muted">{c.wordCount} words | top genre: {c.topGenre} ({Math.round(c.topGenreScore * 100)}%) | timeline markers: {c.timelineMarks}</span>
            </li>
          ))}
        </ul>
      </article>
    </section>
  );
}
