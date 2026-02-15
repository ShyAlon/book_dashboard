import { DashboardData } from "../types";

type Props = {
  data: DashboardData;
};

type ErrorRow = {
  stage: string;
  type: string;
  message: string;
  count: number;
};

function pct(value: number | null): string {
  if (value === null || Number.isNaN(value)) return "n/a";
  return `${(value * 100).toFixed(2)}%`;
}

function metricClass(value: number | null, riskThreshold: number): string {
  if (value === null || Number.isNaN(value)) return "";
  return value >= riskThreshold ? "text-risk" : "text-good";
}

function countWithDupEvidence(data: DashboardData): number {
  return data.aiReport.windows.filter((w) => (w.signals?.duplication?.evidence?.length ?? 0) > 0).length;
}

function countLongDuplicateSpans(data: DashboardData): number {
  return data.aiReport.windows.filter((w) =>
    (w.top_evidence ?? []).some((e) => e.type === "duplication" && e.summary.toLowerCase().includes("long duplicate span")),
  ).length;
}

export function AITab({ data }: Props) {
  const ai = data.aiReport;
  const pDoc = ai.p_ai_doc;
  const pMax = ai.p_ai_max;
  const coverage = ai.ai_coverage_est;
  const confidence = ai.confidence_doc;
  const likelyAI =
    (pMax ?? 0) >= 0.85 ||
    ai.flags.includes("ai_chunk_detected") ||
    ai.flags.includes("widespread_ai_signal") ||
    ((pDoc ?? 0) >= 0.75 && (coverage ?? 0) >= 0.20);
  const dupWindows = countWithDupEvidence(data);
  const longDupWindows = countLongDuplicateSpans(data);
  const groupedErrors = ai.errors.reduce<ErrorRow[]>((acc, err) => {
    const key = `${err.stage}|${err.type}|${err.message}`;
    const found = acc.find((x) => `${x.stage}|${x.type}|${x.message}` === key);
    if (found) {
      found.count += 1;
      return acc;
    }
    acc.push({ stage: err.stage, type: err.type, message: err.message, count: 1 });
    return acc;
  }, []);

  return (
    <section className="panel-grid">
      <article className="panel">
        <h2>AI Likelihood</h2>
        <ul className="list">
          <li><strong>Likely AI Generated:</strong> <span className={likelyAI ? "text-risk" : "text-good"}>{likelyAI ? "Yes" : "No"}</span></li>
          <li><strong>Document AI Probability:</strong> <span className={metricClass(pDoc, 0.5)}>{pct(pDoc)}</span></li>
          <li><strong>AI Coverage Estimate:</strong> <span className={metricClass(coverage, 0.35)}>{pct(coverage)}</span></li>
          <li><strong>Max Window AI Probability:</strong> <span className={metricClass(pMax, 0.85)}>{pct(pMax)}</span></li>
          <li><strong>Document Confidence:</strong> <span className={metricClass(confidence, 0.5)}>{pct(confidence)}</span></li>
          <li><strong>Windows With Duplication Evidence:</strong> <span className={dupWindows > 0 ? "text-risk" : "text-good"}>{dupWindows}</span></li>
          <li><strong>Windows With Long Duplicate Span:</strong> <span className={longDupWindows > 0 ? "text-risk" : "text-good"}>{longDupWindows}</span></li>
          <li><strong>Windows Analyzed:</strong> <span className="text-good">{ai.windows.length}</span></li>
          <li><strong>Detector Errors:</strong> <span className={groupedErrors.length > 0 ? "text-risk" : "text-good"}>{groupedErrors.length}</span></li>
        </ul>
      </article>

      <article className="panel">
        <h2>Detection Flags</h2>
        {ai.flags.length > 0 ? (
          <ul className="list">
            {ai.flags.map((flag, i) => (
              <li key={`${flag}-${i}`} className="text-risk">{flag}</li>
            ))}
          </ul>
        ) : (
          <p className="text-good">No AI-risk flags detected.</p>
        )}
        {groupedErrors.length > 0 && (
          <>
            <h3>Signal Errors</h3>
            <ul className="list">
              {groupedErrors.slice(0, 8).map((err, i) => (
                <li key={`${err.stage}-${i}`} className="text-risk">{`${err.stage}: ${err.type} (${err.message})${err.count > 1 ? ` x${err.count}` : ""}`}</li>
              ))}
            </ul>
          </>
        )}
      </article>
    </section>
  );
}
