package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
}

func NewApp() *App {
	return &App{data: backend.DashboardData{}, services: newServiceManager()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.EventsEmit(a.ctx, "analysis_progress", map[string]any{"percent": 0, "stage": "SETUP", "detail": "initializing"})
	a.services.Start(a.ctx)
	a.data = backend.InitialDashboard()
	a.applySystemDiagnostics(&a.data)
}

func (a *App) shutdown(context.Context) {
	a.services.Stop()
}

func (a *App) GetDashboard() backend.DashboardData {
	a.applySystemDiagnostics(&a.data)
	return a.data
}

func (a *App) GetServiceDiagnostics() backend.SystemDiagnostics {
	return a.services.Snapshot()
}

func (a *App) InstallMissingDependencies() backend.SystemDiagnostics {
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
		return data
	}
	if a.ctx != nil {
		a.services.EnsureReady(a.ctx)
	} else {
		a.services.EnsureReady(nil)
	}
	a.data = backend.BuildDashboard("Pasted Excerpt", "source.txt", []byte(trimmed), trimmed, a.emitProgress)
	a.applySystemDiagnostics(&a.data)
	return a.data
}

func (a *App) AnalyzeFile(path string) backend.DashboardData {
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
	return a.data
}

func (a *App) PickAndAnalyzeFile() backend.DashboardData {
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
		return a.data
	}
	if strings.TrimSpace(selected) == "" {
		return a.GetDashboard()
	}
	return a.AnalyzeFile(selected)
}

func (a *App) ExtractTimelineMarkers(paragraph string) []string {
	return timeline.ExtractMarkers(paragraph)
}

func (a *App) emitProgress(percent int, stage, detail string) {
	if os.Getenv("MHD_TRACE_PROGRESS") == "1" {
		fmt.Printf("%s [PROGRESS] %3d%% [%s] %s\n", time.Now().Format("15:04:05.000"), percent, stage, detail)
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
