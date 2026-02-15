package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"book_dashboard/desktop/backend"
	"book_dashboard/internal/workspace"
)

type logArchive struct {
	mu          sync.Mutex
	rootDir     string
	runsDir     string
	sessionFile string
}

type runSnapshot struct {
	CapturedAt string                `json:"captured_at"`
	Trigger    string                `json:"trigger"`
	Dashboard  backend.DashboardData `json:"dashboard"`
}

func newLogArchive() (*logArchive, error) {
	workspaceRoot, err := workspace.EnsureDefault()
	if err != nil {
		return nil, err
	}
	rootDir := filepath.Join(workspaceRoot, "logs")
	runsDir := filepath.Join(rootDir, "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create runs dir: %w", err)
	}
	sessionFile := filepath.Join(rootDir, "session-"+time.Now().Format("20060102-150405")+".log")
	a := &logArchive{
		rootDir:     rootDir,
		runsDir:     runsDir,
		sessionFile: sessionFile,
	}
	a.appendLine("INFO", "BOOT", "log archive initialized", rootDir)
	return a, nil
}

func (a *logArchive) RootDir() string {
	if a == nil {
		return ""
	}
	return a.rootDir
}

func (a *logArchive) appendLine(level, stage, message, detail string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	line := fmt.Sprintf("[%s] [%s] [%s] %s", time.Now().Format("15:04:05.000"), level, stage, message)
	if strings.TrimSpace(detail) != "" {
		line += " | " + detail
	}
	line += "\n"
	f, err := os.OpenFile(a.sessionFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

func (a *logArchive) appendProgress(percent int, stage, detail string) {
	a.appendLine("ANALYSIS", stage, fmt.Sprintf("progress %d%%", percent), detail)
}

func (a *logArchive) appendServiceTrace(t backend.ServiceTrace) {
	a.appendLine(t.Level, "SETUP", t.Message, t.Detail)
}

func (a *logArchive) appendDashboardLogs(lines []backend.LogLine) {
	for _, l := range lines {
		a.appendLine(l.Level, l.Stage, l.Message, l.Detail)
	}
}

func (a *logArchive) persistRunSnapshot(trigger string, data backend.DashboardData) (string, error) {
	if a == nil {
		return "", fmt.Errorf("log archive unavailable")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	name := time.Now().Format("20060102-150405")
	runID := sanitizeForFilename(strings.TrimSpace(data.RunStats.RunID))
	if runID != "" {
		name += "-" + runID
	}
	trigger = sanitizeForFilename(trigger)
	if trigger != "" {
		name += "-" + trigger
	}
	path := filepath.Join(a.runsDir, name+".json")
	snap := runSnapshot{
		CapturedAt: time.Now().Format(time.RFC3339),
		Trigger:    trigger,
		Dashboard:  data,
	}
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal run snapshot: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", fmt.Errorf("write run snapshot: %w", err)
	}
	return path, nil
}

func (a *logArchive) exportZip(dest string) error {
	if a == nil {
		return fmt.Errorf("log archive unavailable")
	}
	if strings.TrimSpace(dest) == "" {
		return fmt.Errorf("destination path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create destination dir: %w", err)
	}
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer out.Close()

	zipWriter := zip.NewWriter(out)
	defer zipWriter.Close()

	err = filepath.Walk(a.rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(a.rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		w, err := zipWriter.Create(rel)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = w.Write(raw)
		return err
	})
	if err != nil {
		return fmt.Errorf("collect log files: %w", err)
	}
	return nil
}

func sanitizeForFilename(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	out = strings.ReplaceAll(out, "--", "-")
	return out
}
