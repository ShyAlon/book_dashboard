import { DashboardData } from "../types";

type Props = { data: DashboardData };

export function HeaderMetrics({ data }: Props) {
  return (
    <>
      <header className="mhd-header">
        <div>
          <h1>{data.bookTitle}</h1>
          <p>{data.wordCount.toLocaleString()} words</p>
          <p>{data.chapterCount} chapters detected</p>
          <p className="project-path">{data.projectLocation}</p>
        </div>
        <div className={`score-pill ${data.mhdScore >= 70 ? "healthy" : "risk"}`}>MHD Score: {data.mhdScore}</div>
      </header>

      <section className={`run-banner ${data.runStats.status === "DONE" ? "ok" : "pending"}`}>
        <span>{data.runStats.lastAction || "Ready"}</span>
        <span>{data.runStats.status || "IDLE"}</span>
        <span>{data.runStats.runId}</span>
      </section>

      <section className={`run-banner ${data.system.overall === "READY" ? "ok" : "pending"}`}>
        <span>Services: {data.system.overall}</span>
        <span>Ollama: {data.system.ollama.ready ? "ready" : "not ready"}</span>
        <span>LanguageTool: {data.system.languageTool.ready ? "ready" : "not ready"}</span>
      </section>
      {(data.system.ollama.lastError || data.system.languageTool.lastError) ? (
        <section className="panel">
          <h2>Service Errors</h2>
          {data.system.ollama.lastError ? (
            <p className="text-risk">
              <strong>Ollama:</strong> {data.system.ollama.lastError}
              {data.system.ollama.installCommand ? <><br /><code>{data.system.ollama.installCommand}</code></> : null}
            </p>
          ) : null}
          {data.system.languageTool.lastError ? (
            <p className="text-risk">
              <strong>LanguageTool:</strong> {data.system.languageTool.lastError}
              {data.system.languageTool.installCommand ? <><br /><code>{data.system.languageTool.installCommand}</code></> : null}
            </p>
          ) : null}
        </section>
      ) : null}

      <section className="run-metrics">
        <div className="metric"><label>Chapters</label><strong>{data.runStats.chapterCount}</strong></div>
        <div className="metric"><label>Segments</label><strong>{data.runStats.segmentCount}</strong></div>
        <div className="metric"><label>Timeline Markers</label><strong>{data.runStats.timelineCount}</strong></div>
        <div className="metric"><label>Contradictions</label><strong>{data.runStats.contradictionCount}</strong></div>
        <div className="metric"><label>Slop Flags</label><strong>{data.runStats.slopFlagCount}</strong></div>
      </section>
    </>
  );
}
