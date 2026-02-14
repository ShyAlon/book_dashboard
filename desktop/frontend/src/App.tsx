import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import "vis-timeline/styles/vis-timeline-graph2d.css";
import { AnalyzeExcerpt, AnalyzeFile, GetDashboard, InstallMissingDependencies, PickAndAnalyzeFile } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { AnalysisForms } from "./components/AnalysisForms";
import { HeaderMetrics } from "./components/HeaderMetrics";
import { LiveConsole } from "./components/LiveConsole";
import { HealthTab } from "./tabs/HealthTab";
import { LanguageTab } from "./tabs/LanguageTab";
import { MarketTab } from "./tabs/MarketTab";
import { StructureTab } from "./tabs/StructureTab";
import { DictionaryTab } from "./tabs/DictionaryTab";
import { DashboardData, emptyData, LogFilter, LogLine, TabName } from "./types";
import "./App.css";

const STARTUP_STAGE = "SETUP";

type BootWindow = Window & {
  __mhdBootHide?: () => void;
  __mhdBootUpdate?: (action: string, subphase: string, detail: string) => void;
};

function normalizeDashboard(input: unknown): DashboardData {
  const next = (input ?? {}) as Partial<DashboardData>;
  const system = (next.system ?? {}) as Partial<DashboardData["system"]>;
  const runStats = (next.runStats ?? {}) as Partial<DashboardData["runStats"]>;
  return {
    ...emptyData,
    ...next,
    system: {
      ...emptyData.system,
      ...system,
      ollama: { ...emptyData.system.ollama, ...(system.ollama ?? {}) },
      languageTool: { ...emptyData.system.languageTool, ...(system.languageTool ?? {}) },
      traces: Array.isArray(system.traces) ? system.traces : [],
    },
    runStats: {
      ...emptyData.runStats,
      ...runStats,
    },
    logs: Array.isArray(next.logs) ? next.logs : [],
    healthIssues: Array.isArray(next.healthIssues) ? next.healthIssues : [],
    contradictions: Array.isArray(next.contradictions) ? next.contradictions : [],
  };
}

function isInitializationComplete(next: DashboardData): boolean {
  return !next.system.initializing && next.system.overall !== "IDLE";
}

function mergeUniqueLogs(existing: LogLine[], incoming: LogLine[]): LogLine[] {
  if (incoming.length === 0) return existing;
  const seen = new Set(existing.map((l) => `${l.time}|${l.level}|${l.stage}|${l.message}|${l.detail}`));
  const merged = [...existing];
  for (const line of incoming) {
    const key = `${line.time}|${line.level}|${line.stage}|${line.message}|${line.detail}`;
    if (!seen.has(key)) {
      merged.push(line);
      seen.add(key);
    }
  }
  return merged;
}

function toStartupLogs(data: DashboardData): LogLine[] {
  return data.system.traces.map((t) => ({
    time: t.time,
    level: t.level,
    stage: STARTUP_STAGE,
    message: t.message,
    detail: t.detail,
  }));
}

function formatClock(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60).toString().padStart(2, "0");
  const seconds = (totalSeconds % 60).toString().padStart(2, "0");
  return `${minutes}:${seconds}`;
}

function App() {
  const overallStartedAtRef = useRef<number>(Date.now());
  const phaseStartedAtRef = useRef<number>(Date.now());
  const startupConsoleRef = useRef<HTMLDivElement>(null);
  const [tab, setTab] = useState<TabName>("health");
  const [data, setData] = useState<DashboardData>(emptyData);
  const [initComplete, setInitComplete] = useState(false);
  const [startupLogs, setStartupLogs] = useState<LogLine[]>([
    {
      time: new Date().toLocaleTimeString("en-US", { hour12: false }),
      level: "INFO",
      stage: STARTUP_STAGE,
      message: "Initialization started",
      detail: "initializing",
    },
  ]);
  const [excerpt, setExcerpt] = useState("");
  const [filePath, setFilePath] = useState("");
  const [loading, setLoading] = useState(false);
  const [logFilter, setLogFilter] = useState<LogFilter>("ALL");
  const [logQuery, setLogQuery] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);
  const [liveProgressLogs, setLiveProgressLogs] = useState<LogLine[]>([]);
  const [chapterSubtasks, setChapterSubtasks] = useState<string[]>([]);
  const [progress, setProgress] = useState({ percent: 0, stage: "IDLE", detail: "" });
  const [selectedIssue, setSelectedIssue] = useState<number>(-1);
  const [phaseElapsedSeconds, setPhaseElapsedSeconds] = useState(0);
  const [overallElapsedSeconds, setOverallElapsedSeconds] = useState(0);
  const [installingDeps, setInstallingDeps] = useState(false);
  const consoleRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const id = window.setInterval(() => {
      setPhaseElapsedSeconds(Math.floor((Date.now() - phaseStartedAtRef.current) / 1000));
      setOverallElapsedSeconds(Math.floor((Date.now() - overallStartedAtRef.current) / 1000));
    }, 1000);
    return () => {
      window.clearInterval(id);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    let pollId = 0;

    const syncDashboard = async () => {
      const next = normalizeDashboard(await GetDashboard());
      if (cancelled) return;
      setData(next);
      setStartupLogs((prev) => mergeUniqueLogs(prev, toStartupLogs(next)));
      const boot = window as BootWindow;
      boot.__mhdBootUpdate?.("initializing", "dependency checks", `services=${next.system.overall}`);
      if (isInitializationComplete(next)) {
        setInitComplete(true);
        boot.__mhdBootHide?.();
        if (pollId !== 0) {
          window.clearInterval(pollId);
        }
      }
    };

    void syncDashboard();
    pollId = window.setInterval(() => {
      void syncDashboard();
    }, 800);

    return () => {
      cancelled = true;
      if (pollId !== 0) {
        window.clearInterval(pollId);
      }
    };
  }, []);

  useEffect(() => {
    const off = EventsOn("analysis_progress", (payload: { percent: number; stage: string; detail: string }) => {
      if (!payload) return;
      const stage = (payload.stage ?? "RUNNING").toUpperCase();
      const detail = payload.detail ?? "";
      setProgress({
        percent: payload.percent ?? 0,
        stage,
        detail,
      });

      if (stage === STARTUP_STAGE && detail.trim() !== "") {
        (window as BootWindow).__mhdBootUpdate?.("initializing", detail, detail);
        const line: LogLine = {
          time: new Date().toLocaleTimeString("en-US", { hour12: false }),
          level: "INFO",
          stage: STARTUP_STAGE,
          message: stage,
          detail,
        };
        setStartupLogs((prev) => mergeUniqueLogs(prev, [line]));
        return;
      }

      if (detail.trim() !== "") {
        const line: LogLine = {
          time: new Date().toLocaleTimeString("en-US", { hour12: false }),
          level: stage === "DONE" ? "INFO" : "ANALYSIS",
          stage,
          message: detail,
          detail: "",
        };
        setLiveProgressLogs((prev) => mergeUniqueLogs(prev, [line]).slice(-250));
      }

      if (stage === "CHAPTER" && detail.trim() !== "") {
        setChapterSubtasks((prev) => {
          if (prev[prev.length - 1] === detail) return prev;
          const next = [...prev, detail];
          return next.slice(-8);
        });
      }
    });
    return () => {
      off();
    };
  }, []);

  useEffect(() => {
    const off = EventsOn("service_trace", (payload: { time: string; level: string; message: string; detail: string }) => {
      if (!payload) return;
      const line: LogLine = {
        time: payload.time || new Date().toLocaleTimeString("en-US", { hour12: false }),
        level: payload.level || "INFO",
        stage: STARTUP_STAGE,
        message: payload.message || "service trace",
        detail: payload.detail || "",
      };
      (window as BootWindow).__mhdBootUpdate?.("initializing", [line.message, line.detail].filter(Boolean).join(": "), [line.message, line.detail].filter(Boolean).join(": "));
      setStartupLogs((prev) => mergeUniqueLogs(prev, [line]));
      setProgress({ percent: 1, stage: STARTUP_STAGE, detail: [line.message, line.detail].filter(Boolean).join(": ") });
    });
    return () => {
      off();
    };
  }, []);

  useEffect(() => {
    if (autoScroll && consoleRef.current) {
      consoleRef.current.scrollTop = consoleRef.current.scrollHeight;
    }
  }, [data.logs, autoScroll]);

  useEffect(() => {
    if (startupConsoleRef.current) {
      startupConsoleRef.current.scrollTop = startupConsoleRef.current.scrollHeight;
    }
  }, [startupLogs]);

  const filteredLogs = useMemo(() => {
    const serviceLogs = data.system.traces.map((t) => ({
      time: t.time,
      level: t.level,
      stage: "SETUP",
      message: t.message,
      detail: t.detail,
    }));
    const merged = mergeUniqueLogs([...serviceLogs, ...data.logs], liveProgressLogs);
    return merged.filter((line) => {
      if (logFilter !== "ALL" && line.level !== logFilter) {
        return false;
      }
      if (!logQuery.trim()) {
        return true;
      }
      const q = logQuery.toLowerCase();
      return [line.message, line.detail, line.stage, line.level].join(" ").toLowerCase().includes(q);
    });
  }, [data.logs, data.system.traces, liveProgressLogs, logFilter, logQuery]);

  const onAnalyzeExcerpt = async (e: FormEvent) => {
    e.preventDefault();
    setLiveProgressLogs([]);
    setChapterSubtasks([]);
    setProgress({ percent: 0, stage: "ANALYSIS", detail: "Starting excerpt analysis..." });
    setLoading(true);
    try {
      const next = await AnalyzeExcerpt(excerpt);
      setData(next as unknown as DashboardData);
      setSelectedIssue(-1);
    } finally {
      setLoading(false);
    }
  };

  const onAnalyzeFile = async (e: FormEvent) => {
    e.preventDefault();
    setLiveProgressLogs([]);
    setChapterSubtasks([]);
    setProgress({ percent: 0, stage: "ANALYSIS", detail: "Starting file analysis..." });
    setLoading(true);
    try {
      const next = await AnalyzeFile(filePath);
      setData(next as unknown as DashboardData);
      setSelectedIssue(-1);
    } finally {
      setLoading(false);
    }
  };

  const onPickAndAnalyze = async () => {
    setLiveProgressLogs([]);
    setChapterSubtasks([]);
    setProgress({ percent: 0, stage: "ANALYSIS", detail: "Opening file picker..." });
    setLoading(true);
    try {
      const next = await PickAndAnalyzeFile();
      setData(next as unknown as DashboardData);
      setSelectedIssue(-1);
    } finally {
      setLoading(false);
    }
  };

  const centerPhase = useMemo(() => {
    if (!initComplete) return "initializing";
    if (loading) return "analysis";
    return "idle";
  }, [initComplete, loading]);

  const centerSubphase = useMemo(() => {
    if (!initComplete) {
      if (progress.detail.trim()) return progress.detail;
      if (startupLogs.length > 0) {
        const last = startupLogs[startupLogs.length - 1];
        return [last.message, last.detail].filter(Boolean).join(": ");
      }
      return "starting up";
    }
    if (loading) {
      if (progress.detail.trim()) return progress.detail;
      if (progress.stage.trim()) return progress.stage;
      return "running analysis";
    }
    const action = data.runStats.lastAction?.trim();
    if (action) return action;
    return "ready";
  }, [loading, progress.detail, progress.stage, data.runStats.lastAction]);

  useEffect(() => {
    phaseStartedAtRef.current = Date.now();
    setPhaseElapsedSeconds(0);
  }, [centerPhase]);

  const missingStatuses = useMemo(() => {
    return [data.system.ollama, data.system.languageTool].filter((s) => s.missing);
  }, [data.system.ollama, data.system.languageTool]);

  const onInstallMissingDependencies = async () => {
    if (installingDeps) return;
    const details = missingStatuses.map((s) => `${s.name}: ${s.installCommand || s.installHint || "missing dependency"}`).join("\n");
    const approved = window.confirm(`Install missing dependencies now?\n\n${details}`);
    if (!approved) return;

    setInstallingDeps(true);
    setProgress({ percent: 1, stage: STARTUP_STAGE, detail: "Installing missing dependencies..." });
    try {
      const diag = await InstallMissingDependencies();
      setData((prev) => normalizeDashboard({ ...prev, system: diag }));
    } finally {
      setInstallingDeps(false);
    }
  };

  const showCenterStatus = !initComplete || loading;

  return (
    <div className="mhd-app">
      {showCenterStatus ? (
        <section className="center-status" aria-live="polite">
          <div className="center-status-card">
            <p className="center-status-label">Phase</p>
            <p className="center-status-action">{centerPhase}</p>
            <p className="center-status-label">Sub-phase</p>
            <p className="center-status-subphase">{centerSubphase}</p>
            {loading && chapterSubtasks.length > 0 ? (
              <section className="center-status-subtasks">
                <p className="center-status-label">Chapter Sub-Tasks</p>
                <ul>
                  {chapterSubtasks.map((entry, idx) => (
                    <li key={`${entry}-${idx}`}>{entry}</li>
                  ))}
                </ul>
              </section>
            ) : null}
            <p className="center-status-time">Phase {formatClock(phaseElapsedSeconds)}</p>
            <p className="center-status-time-secondary">Overall {formatClock(overallElapsedSeconds)}</p>
          </div>
        </section>
      ) : null}

      {!initComplete ? (
        <aside className="startup-console panel" ref={startupConsoleRef}>
          <h2>Startup Console</h2>
          <p className="muted">Initialization logs until the app is ready.</p>
          {missingStatuses.length > 0 ? (
            <section className="panel startup-missing">
              <h3>Missing Dependencies</h3>
              {missingStatuses.map((s) => (
                <p key={s.name} className="text-risk">
                  <strong>{s.name}:</strong> {s.installHint || s.lastError || "Missing dependency"}
                  <br />
                  <code>{s.installCommand || "No install command available"}</code>
                </p>
              ))}
              <button type="button" onClick={onInstallMissingDependencies} disabled={installingDeps}>
                {installingDeps ? "Installing..." : "Install Missing Dependencies"}
              </button>
            </section>
          ) : null}
          <ul>
            {startupLogs.map((line, idx) => (
              <li key={`${line.time}-${line.level}-${idx}`} className={`log-${line.level.toLowerCase()}`}>
                <div>[{line.time}] [{line.level}] [{line.stage}] {line.message}</div>
                {line.detail ? <div className="log-detail">{line.detail}</div> : null}
              </li>
            ))}
          </ul>
        </aside>
      ) : null}

      {initComplete ? <HeaderMetrics data={data} /> : null}

      {initComplete ? (
        <div className="mhd-layout">
        <main className="mhd-main">
          <AnalysisForms
            excerpt={excerpt}
            setExcerpt={setExcerpt}
            filePath={filePath}
            setFilePath={setFilePath}
            loading={loading}
            onAnalyzeExcerpt={onAnalyzeExcerpt}
            onAnalyzeFile={onAnalyzeFile}
            onPickAndAnalyze={onPickAndAnalyze}
          />

          {loading ? (
            <section className="progress-wrap">
              <div className="progress-head">
                <strong>{progress.stage}</strong>
                <span>{progress.percent}%</span>
              </div>
              <div className="progress-bar">
                <div className="progress-fill" style={{ width: `${progress.percent}%` }} />
              </div>
              <p className="muted">{progress.detail}</p>
            </section>
          ) : null}

          <nav className="tabs">
            <button className={tab === "health" ? "active" : ""} onClick={() => setTab("health")}>Health</button>
            <button className={tab === "structure" ? "active" : ""} onClick={() => setTab("structure")}>Structure</button>
            <button className={tab === "market" ? "active" : ""} onClick={() => setTab("market")}>Market</button>
            <button className={tab === "language" ? "active" : ""} onClick={() => setTab("language")}>Language</button>
            <button className={tab === "dictionary" ? "active" : ""} onClick={() => setTab("dictionary")}>Dictionary</button>
          </nav>

          {tab === "health" && <HealthTab data={data} selectedIssue={selectedIssue} setSelectedIssue={setSelectedIssue} />}
          {tab === "structure" && <StructureTab data={data} />}
          {tab === "market" && <MarketTab data={data} />}
          {tab === "language" && <LanguageTab data={data} />}
          {tab === "dictionary" && <DictionaryTab data={data} />}
        </main>

        <LiveConsole
          consoleRef={consoleRef}
          filteredLogs={filteredLogs}
          logFilter={logFilter}
          setLogFilter={setLogFilter}
          logQuery={logQuery}
          setLogQuery={setLogQuery}
          autoScroll={autoScroll}
          setAutoScroll={setAutoScroll}
        />
        </div>
      ) : null}
    </div>
  );
}

export default App;
