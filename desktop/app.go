package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"book_dashboard/desktop/backend"
	"book_dashboard/internal/ingest"
	"book_dashboard/internal/timeline"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	data     backend.DashboardData
	services *serviceManager
	logs     *logArchive
}

func NewApp() *App {
	return &App{data: backend.DashboardData{}, services: newServiceManager()}
}

func (a *App) startup(ctx context.Context) {
	defer a.recoverFromPanic("startup")
	a.ctx = ctx
	if archive, err := newLogArchive(); err == nil {
		a.logs = archive
	} else {
		fmt.Printf("%s [RISK] [LOGS] Failed to initialize log archive: %v\n", time.Now().Format("15:04:05.000"), err)
	}
	a.services.SetTraceSink(func(t backend.ServiceTrace) {
		if a.logs != nil {
			a.logs.appendServiceTrace(t)
		}
	})
	runtime.EventsEmit(a.ctx, "analysis_progress", map[string]any{"percent": 0, "stage": "SETUP", "detail": "initializing"})
	a.services.Start(a.ctx)
	a.data = backend.InitialDashboard()
	a.applySystemDiagnostics(&a.data)
	a.persistDashboardSnapshot("startup")
}

func (a *App) shutdown(context.Context) {
	a.services.Stop()
}

func (a *App) GetDashboard() backend.DashboardData {
	defer a.recoverFromPanic("GetDashboard")
	a.applySystemDiagnostics(&a.data)
	return a.data
}

func (a *App) GetServiceDiagnostics() backend.SystemDiagnostics {
	defer a.recoverFromPanic("GetServiceDiagnostics")
	return a.services.Snapshot()
}

func (a *App) InstallMissingDependencies() backend.SystemDiagnostics {
	defer a.recoverFromPanic("InstallMissingDependencies")
	diag := a.services.Snapshot()
	pkgSet := map[string]struct{}{}

	for _, status := range []backend.ServiceStatus{diag.Ollama, diag.LanguageTool} {
		if !status.Missing {
			continue
		}
		if pkg := packageForServiceStatus(status); pkg != "" {
			pkgSet[pkg] = struct{}{}
		}
	}

	packages := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	if len(packages) == 0 {
		a.services.trace(a.ctx, "INFO", "Dependency install skipped", "No missing dependencies detected")
		a.applySystemDiagnostics(&a.data)
		return a.services.Snapshot()
	}

	for _, pkg := range packages {
		a.services.trace(a.ctx, "ANALYSIS", "Installing dependency", pkg)
		if err := installWithBrew(pkg); err != nil {
			a.services.trace(a.ctx, "RISK", "Dependency install failed", fmt.Sprintf("%s: %v", pkg, err))
		} else {
			a.services.trace(a.ctx, "INFO", "Dependency installed", pkg)
		}
	}

	if a.ctx != nil {
		a.services.EnsureReady(a.ctx)
	} else {
		a.services.EnsureReady(nil)
	}
	a.applySystemDiagnostics(&a.data)
	return a.services.Snapshot()
}

func (a *App) AnalyzeExcerpt(text string) backend.DashboardData {
	defer a.recoverFromPanic("AnalyzeExcerpt")
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		data := a.GetDashboard()
		data.Logs = append(data.Logs, backend.LogLine{
			Time:    time.Now().Format("15:04:05.000"),
			Level:   "RISK",
			Stage:   "INGEST",
			Message: "Analyze Excerpt ignored: empty text",
			Detail:  "Paste text before running excerpt analysis.",
		})
		a.data = data
		a.persistDashboardSnapshot("analyze_excerpt_empty")
		return data
	}
	if a.ctx != nil {
		a.services.EnsureReady(a.ctx)
	} else {
		a.services.EnsureReady(nil)
	}
	a.data = backend.BuildDashboard("Pasted Excerpt", "source.txt", []byte(trimmed), trimmed, a.emitProgress)
	a.applySystemDiagnostics(&a.data)
	a.persistDashboardSnapshot("analyze_excerpt")
	return a.data
}

func (a *App) AnalyzeFile(path string) backend.DashboardData {
	defer a.recoverFromPanic("AnalyzeFile")
	path = strings.TrimSpace(path)
	if path == "" {
		data := a.GetDashboard()
		data.Logs = append(data.Logs, backend.LogLine{
			Time:    time.Now().Format("15:04:05.000"),
			Level:   "RISK",
			Stage:   "INGEST",
			Message: "Analyze File ignored: empty path",
			Detail:  "Provide an absolute .docx or .pdf path or use Pick File.",
		})
		a.data = data
		a.persistDashboardSnapshot("analyze_file_empty")
		return data
	}
	if _, err := os.Stat(path); err != nil {
		data := a.GetDashboard()
		data.Logs = append(data.Logs, backend.LogLine{
			Time:    time.Now().Format("15:04:05.000"),
			Level:   "RISK",
			Stage:   "INGEST",
			Message: "Analyze File failed: path not found",
			Detail:  path,
		})
		a.data = data
		a.persistDashboardSnapshot("analyze_file_not_found")
		return data
	}

	parsed, err := ingest.ParseFile(path)
	if err != nil {
		data := backend.BuildDashboard("Ingestion Failure", "", nil, backend.DefaultDemoText, a.emitProgress)
		data.Logs = append(data.Logs, backend.LogLine{
			Time:    time.Now().Format("15:04:05.000"),
			Level:   "RISK",
			Stage:   "INGEST",
			Message: "file parse failed",
			Detail:  err.Error(),
		})
		a.data = data
		a.persistDashboardSnapshot("analyze_file_parse_failed")
		return a.data
	}
	if a.ctx != nil {
		a.services.EnsureReady(a.ctx)
	} else {
		a.services.EnsureReady(nil)
	}
	a.emitProgress(10, "INGEST", "File parsed, starting analysis")
	a.data = backend.BuildDashboard(parsed.Title, filepath.Base(parsed.SourcePath), parsed.SourceBytes, parsed.Text, a.emitProgress)
	a.applySystemDiagnostics(&a.data)
	a.persistDashboardSnapshot("analyze_file")
	return a.data
}

func (a *App) PickAndAnalyzeFile() backend.DashboardData {
	defer a.recoverFromPanic("PickAndAnalyzeFile")
	if a.ctx == nil {
		data := a.GetDashboard()
		data.Logs = append(data.Logs, backend.LogLine{
			Time:    time.Now().Format("15:04:05.000"),
			Level:   "RISK",
			Stage:   "INGEST",
			Message: "File picker unavailable",
			Detail:  "UI context is not initialized.",
		})
		a.data = data
		a.persistDashboardSnapshot("pick_file_unavailable")
		return data
	}
	selected, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Manuscript",
		Filters: []runtime.FileFilter{
			{DisplayName: "Manuscript Files", Pattern: "*.docx;*.pdf"},
			{DisplayName: "Word Document", Pattern: "*.docx"},
			{DisplayName: "PDF", Pattern: "*.pdf"},
		},
	})
	if err != nil {
		data := a.GetDashboard()
		data.Logs = append(data.Logs, backend.LogLine{
			Time:    time.Now().Format("15:04:05.000"),
			Level:   "RISK",
			Stage:   "INGEST",
			Message: "file picker failed",
			Detail:  err.Error(),
		})
		a.data = data
		a.persistDashboardSnapshot("pick_file_error")
		return a.data
	}
	if strings.TrimSpace(selected) == "" {
		return a.GetDashboard()
	}
	return a.AnalyzeFile(selected)
}

func (a *App) ExtractTimelineMarkers(paragraph string) []string {
	defer a.recoverFromPanic("ExtractTimelineMarkers")
	return timeline.ExtractMarkers(paragraph)
}

func (a *App) emitProgress(percent int, stage, detail string) {
	if os.Getenv("MHD_TRACE_PROGRESS") == "1" {
		fmt.Printf("%s [PROGRESS] %3d%% [%s] %s\n", time.Now().Format("15:04:05.000"), percent, stage, detail)
	}
	if a.logs != nil {
		a.logs.appendProgress(percent, stage, detail)
	}
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "analysis_progress", map[string]any{
		"percent": percent,
		"stage":   stage,
		"detail":  detail,
	})
}

func (a *App) applySystemDiagnostics(data *backend.DashboardData) {
	if data == nil {
		return
	}
	data.System = a.services.Snapshot()
}

func (a *App) persistDashboardSnapshot(trigger string) {
	if a.logs == nil {
		return
	}
	a.applySystemDiagnostics(&a.data)
	a.logs.appendDashboardLogs(a.data.Logs)
	path, err := a.logs.persistRunSnapshot(trigger, a.data)
	if err != nil {
		fmt.Printf("%s [RISK] [LOGS] Failed to persist snapshot: %v\n", time.Now().Format("15:04:05.000"), err)
		return
	}
	a.logs.appendLine("INFO", "LOGS", "Run snapshot persisted", path)
}

func (a *App) ExportLogPackageDialog() {
	defer a.recoverFromPanic("ExportLogPackageDialog")
	if a.ctx == nil {
		return
	}
	if a.logs == nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Export Log Package",
			Message: "Log archive is not initialized.",
		})
		return
	}
	defaultDir := a.logs.RootDir()
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		downloads := filepath.Join(home, "Downloads")
		if stat, statErr := os.Stat(downloads); statErr == nil && stat.IsDir() {
			defaultDir = downloads
		}
	}
	target, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Export Log Package",
		DefaultDirectory: defaultDir,
		DefaultFilename:  "mhd-log-package-" + time.Now().Format("20060102-150405") + ".zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP Archive", Pattern: "*.zip"},
		},
	})
	if err != nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Export Log Package",
			Message: "Could not open save dialog: " + err.Error(),
		})
		return
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	if !strings.HasSuffix(strings.ToLower(target), ".zip") {
		target += ".zip"
	}
	if err := a.logs.exportZip(target); err != nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Export Log Package",
			Message: "Failed to export logs: " + err.Error(),
		})
		return
	}
	a.logs.appendLine("INFO", "LOGS", "Log package exported", target)
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Export Log Package",
		Message: "Log package created at:\n" + target,
	})
}

func (a *App) Quit() {
	defer a.recoverFromPanic("Quit")
	if a.ctx == nil {
		return
	}
	runtime.Quit(a.ctx)
}

func (a *App) ReportClientError(source, message, detail string) {
	line := backend.LogLine{
		Time:    time.Now().Format("15:04:05.000"),
		Level:   "RISK",
		Stage:   "FRONTEND",
		Message: strings.TrimSpace(source + ": " + message),
		Detail:  strings.TrimSpace(detail),
	}
	if line.Message == ":" || line.Message == "" {
		line.Message = "frontend runtime error"
	}
	a.data.Logs = append(a.data.Logs, line)
	if a.logs != nil {
		a.logs.appendLine(line.Level, line.Stage, line.Message, line.Detail)
	}
	a.persistDashboardSnapshot("frontend_error")
}

func (a *App) recoverFromPanic(where string) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("%v", r)
		stack := string(debug.Stack())
		a.ReportClientError("panic:"+where, msg, stack)
		fmt.Printf("%s [RISK] [PANIC] %s: %s\n%s\n", time.Now().Format("15:04:05.000"), where, msg, stack)
	}
}

func packageForServiceStatus(status backend.ServiceStatus) string {
	combined := strings.ToLower(status.LastError + " " + status.Detail + " " + status.InstallHint + " " + status.InstallCommand)
	switch {
	case strings.Contains(combined, "ollama"):
		return "ollama"
	case strings.Contains(combined, "languagetool"):
		return "languagetool"
	case strings.Contains(combined, "java"), strings.Contains(combined, "openjdk"):
		return "openjdk"
	default:
		return ""
	}
}
