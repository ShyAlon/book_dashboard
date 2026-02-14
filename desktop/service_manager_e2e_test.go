package main

import (
	"strings"
	"testing"
)

func TestServiceLifecycleDiagnosticsMissingBinaries_E2E(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("MHD_DISABLE_SYSTEM_BIN_FALLBACK", "1")
	t.Setenv("LANGUAGETOOL_JAR", "")
	t.Setenv("OLLAMA_URL", "http://127.0.0.1:9")
	t.Setenv("LANGUAGETOOL_URL", "http://127.0.0.1:9")
	t.Setenv("OLLAMA_LANGUAGE_MODEL", "llama3")

	sm := newServiceManager()
	sm.EnsureReady(nil)
	snap := sm.Snapshot()

	if snap.Overall == "READY" {
		t.Fatal("expected degraded/idle status when services cannot be started")
	}
	if snap.Ollama.Ready {
		t.Fatal("expected ollama not ready with empty PATH")
	}
	if snap.LanguageTool.Ready {
		t.Fatal("expected language tool not ready with empty PATH and no jar")
	}
	if snap.Ollama.LastError == "" {
		t.Fatal("expected ollama lastError to be populated")
	}
	if snap.LanguageTool.LastError == "" {
		t.Fatal("expected language tool lastError to be populated")
	}
	if len(snap.Traces) == 0 {
		t.Fatal("expected service traces to be populated")
	}

	traceJoined := ""
	for _, tr := range snap.Traces {
		traceJoined += tr.Message + " " + tr.Detail + "\n"
	}
	if !strings.Contains(strings.ToLower(traceJoined), "failed") {
		t.Fatal("expected trace messages to contain startup failure details")
	}
}
