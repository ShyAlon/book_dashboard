import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'

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

window.setInterval(() => {
  if (bootTime) {
    bootTime.textContent = `${Math.floor((Date.now() - bootStartedAt) / 1000)}s`;
  }
}, 1000);

bootAppend("Frontend bootstrap started");
window.addEventListener("error", (e) => {
  const msg = e.error?.message || e.message || "Unknown error";
  bootSetAction("startup error");
  bootSetSubphase("javascript runtime error");
  bootAppend(`ERROR: ${msg}`);
});
window.addEventListener("unhandledrejection", (e) => {
  bootSetAction("startup error");
  bootSetSubphase("promise rejection");
  bootAppend(`PROMISE: ${String(e.reason)}`);
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
  bootSetAction("startup error");
  bootSetSubphase("react render failure");
  bootAppend(`RENDER: ${message}`);
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
