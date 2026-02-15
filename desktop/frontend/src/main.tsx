import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'
import { LogError } from '../wailsjs/runtime/runtime'

const bootScreen = document.getElementById("boot-screen");
const bootAction = document.getElementById("boot-action");
const bootSubphase = document.getElementById("boot-subphase");
const bootTime = document.getElementById("boot-time");
const bootLog = document.getElementById("boot-log");
const bootStartedAt = Date.now();

function bootNow(): string {
  return new Date().toLocaleTimeString("en-US", { hour12: false });
}

function bootAppend(line: string) {
  if (!bootLog) return;
  const li = document.createElement("li");
  li.textContent = `[${bootNow()}] ${line}`;
  bootLog.appendChild(li);
  bootLog.scrollTop = bootLog.scrollHeight;
}

function bootSetAction(action: string) {
  if (bootAction) {
    bootAction.textContent = action || "initializing";
  }
}

function bootSetSubphase(subphase: string) {
  if (bootSubphase) {
    bootSubphase.textContent = subphase || "working";
  }
}

function reportClientError(source: string, message: string, detail: string) {
  try {
    const api = (window as unknown as {
      go?: { main?: { App?: { ReportClientError?: (s: string, m: string, d: string) => Promise<void> } } }
    }).go?.main?.App;
    if (api?.ReportClientError) {
      void api.ReportClientError(source, message, detail);
    } else {
      LogError(`[${source}] ${message} ${detail}`);
    }
  } catch (err) {
    const fallback = err instanceof Error ? `${err.message}\n${err.stack || ""}` : String(err);
    LogError(`[frontend-report-failed] ${fallback}`);
  }
}

window.setInterval(() => {
  if (bootTime) {
    bootTime.textContent = `${Math.floor((Date.now() - bootStartedAt) / 1000)}s`;
  }
}, 1000);

bootAppend("Frontend bootstrap started");
window.addEventListener("error", (e) => {
  const msg = e.error?.message || e.message || "Unknown error";
  const stack = e.error?.stack ? String(e.error.stack) : "";
  const detail = [e.filename ? `file=${e.filename}` : "", Number.isFinite(e.lineno) ? `line=${e.lineno}` : "", Number.isFinite(e.colno) ? `col=${e.colno}` : "", stack].filter(Boolean).join("\n");
  bootSetAction("startup error");
  bootSetSubphase("javascript runtime error");
  bootAppend(`ERROR: ${msg}`);
  reportClientError("window.error", msg, detail);
});
window.addEventListener("unhandledrejection", (e) => {
  const reason = e.reason instanceof Error ? `${e.reason.message}\n${e.reason.stack || ""}` : String(e.reason);
  bootSetAction("startup error");
  bootSetSubphase("promise rejection");
  bootAppend(`PROMISE: ${reason}`);
  reportClientError("window.unhandledrejection", "Unhandled promise rejection", reason);
});

const container = document.getElementById('root')

const root = createRoot(container!)

try {
  bootAppend("React render start");
  root.render(
      <React.StrictMode>
          <App/>
      </React.StrictMode>
  )
} catch (e) {
  const message = e instanceof Error ? e.message : String(e);
  const stack = e instanceof Error ? e.stack || "" : "";
  bootSetAction("startup error");
  bootSetSubphase("react render failure");
  bootAppend(`RENDER: ${message}`);
  reportClientError("react.render", message, stack);
}

if (bootScreen) {
  (window as unknown as { __mhdBootHide?: () => void }).__mhdBootHide = () => {
    bootScreen.classList.add("hidden");
  };
  (window as unknown as { __mhdBootUpdate?: (action: string, subphase: string, detail: string) => void }).__mhdBootUpdate = (action: string, subphase: string, detail: string) => {
    bootSetAction(action);
    bootSetSubphase(subphase);
    if (detail) {
      bootAppend(detail);
    }
  };
}
