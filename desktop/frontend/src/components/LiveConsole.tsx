import { RefObject } from "react";
import { LogFilter, LogLine } from "../types";

type Props = {
  consoleRef: RefObject<HTMLDivElement>;
  filteredLogs: LogLine[];
  logFilter: LogFilter;
  setLogFilter: (v: LogFilter) => void;
  logQuery: string;
  setLogQuery: (v: string) => void;
  autoScroll: boolean;
  setAutoScroll: (v: boolean | ((x: boolean) => boolean)) => void;
};

export function LiveConsole(props: Props) {
  return (
    <aside className="console panel" ref={props.consoleRef}>
      <h2>Live Console</h2>
      <div className="console-controls">
        <select value={props.logFilter} onChange={(e) => props.setLogFilter(e.target.value as LogFilter)}>
          <option value="ALL">All</option>
          <option value="INFO">Info</option>
          <option value="ANALYSIS">Analysis</option>
          <option value="RISK">Risk</option>
        </select>
        <input value={props.logQuery} onChange={(e) => props.setLogQuery(e.target.value)} placeholder="Search logs..." />
        <button type="button" className="ghost mini" onClick={() => props.setAutoScroll((v) => !v)}>
          {props.autoScroll ? "Auto" : "Paused"}
        </button>
      </div>
      <ul>
        {props.filteredLogs.map((line, idx) => (
          <li key={`${line.time}-${line.level}-${idx}`} className={`log-${line.level.toLowerCase()}`}>
            <div>[{line.time}] [{line.level}] [{line.stage}] {line.message}</div>
            {line.detail ? <div className="log-detail">{line.detail}</div> : null}
          </li>
        ))}
      </ul>
    </aside>
  );
}
