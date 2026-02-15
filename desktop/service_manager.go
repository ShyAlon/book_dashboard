package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"book_dashboard/desktop/backend"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type serviceManager struct {
	mu sync.Mutex

	started      bool
	ready        bool
	initializing bool

	ollamaProc       *managedProcess
	languageToolProc *managedProcess

	ollamaStatus       backend.ServiceStatus
	languageToolStatus backend.ServiceStatus
	traces             []backend.ServiceTrace
	traceSink          func(backend.ServiceTrace)
}

type managedProcess struct {
	name string
	cmd  *exec.Cmd
}

func newServiceManager() *serviceManager {
	return &serviceManager{
		ollamaStatus:       backend.ServiceStatus{Name: "ollama"},
		languageToolStatus: backend.ServiceStatus{Name: "languagetool"},
		traces:             make([]backend.ServiceTrace, 0, 400),
	}
}

func (s *serviceManager) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()
	go s.ensureReadyInternal(ctx)
}

func (s *serviceManager) EnsureReady(ctx context.Context) {
	s.mu.Lock()
	if s.ready || s.initializing {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	s.ensureReadyInternal(ctx)
}

func (s *serviceManager) ensureReadyInternal(ctx context.Context) {
	s.mu.Lock()
	if s.ready || s.initializing {
		s.mu.Unlock()
		return
	}
	s.initializing = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.initializing = false
		s.ready = s.ollamaStatus.Ready && s.languageToolStatus.Ready
		s.mu.Unlock()
	}()

	ltURL := getenv("LANGUAGETOOL_URL", "http://localhost:8010/v2/check")
	ollamaURL := getenv("OLLAMA_URL", "http://127.0.0.1:11434")
	model := getenv("OLLAMA_LANGUAGE_MODEL", "llama3.1:8b")
	genreModel := getenv("OLLAMA_GENRE_MODEL", model)

	s.trace(ctx, "INFO", "Service lifecycle start", "initializing dependencies")

	// LanguageTool
	if isHTTPAlive(ltURL, 2*time.Second) {
		s.updateLanguageTool(true, true, "using existing endpoint", "")
		s.trace(ctx, "INFO", "LanguageTool ready", ltURL)
	} else {
		cmd, err := startLanguageTool()
		if err != nil {
			s.updateLanguageTool(false, false, "startup failed", err.Error())
			s.trace(ctx, "RISK", "LanguageTool start failed", err.Error())
		} else {
			s.mu.Lock()
			s.languageToolProc = cmd
			s.mu.Unlock()
			s.trace(ctx, "ANALYSIS", "LanguageTool process started", "waiting for health endpoint")
			waitForHTTP(ltURL, 35*time.Second)
			if isHTTPAlive(ltURL, 2*time.Second) {
				s.updateLanguageTool(true, true, "started by app", "")
				s.trace(ctx, "INFO", "LanguageTool ready", ltURL)
			} else {
				s.updateLanguageTool(true, false, "process started but endpoint unreachable", "timeout")
				s.trace(ctx, "RISK", "LanguageTool endpoint unreachable", ltURL)
			}
		}
	}

	// Ollama server
	tagsURL := strings.TrimSuffix(ollamaURL, "/") + "/api/tags"
	if isHTTPAlive(tagsURL, 2*time.Second) {
		s.updateOllama(true, true, "using existing endpoint", "")
		s.trace(ctx, "INFO", "Ollama ready", tagsURL)
	} else {
		cmd, err := startOllamaServe()
		if err != nil {
			s.updateOllama(false, false, "startup failed", err.Error())
			s.trace(ctx, "RISK", "Ollama start failed", err.Error())
		} else {
			s.mu.Lock()
			s.ollamaProc = cmd
			s.mu.Unlock()
			s.trace(ctx, "ANALYSIS", "Ollama process started", "waiting for tags endpoint")
			waitForHTTP(tagsURL, 30*time.Second)
			if isHTTPAlive(tagsURL, 2*time.Second) {
				s.updateOllama(true, true, "started by app", "")
				s.trace(ctx, "INFO", "Ollama ready", tagsURL)
			} else {
				s.updateOllama(true, false, "process started but endpoint unreachable", "timeout")
				s.trace(ctx, "RISK", "Ollama endpoint unreachable", tagsURL)
			}
		}
	}

	// Model lifecycle
	s.mu.Lock()
	ollamaReady := s.ollamaStatus.Ready
	s.mu.Unlock()
	if ollamaReady {
		s.trace(ctx, "ANALYSIS", "Ensuring Ollama language model", model)
		if err := pullModel(model); err != nil {
			s.trace(ctx, "RISK", "Ollama model pull failed", err.Error())
			s.mu.Lock()
			s.ollamaStatus.LastError = err.Error()
			s.mu.Unlock()
		} else {
			s.trace(ctx, "INFO", "Ollama model ready", model)
		}
		if genreModel != model {
			s.trace(ctx, "ANALYSIS", "Ensuring Ollama genre model", genreModel)
			if err := pullModel(genreModel); err != nil {
				s.trace(ctx, "RISK", "Ollama genre model pull failed", err.Error())
				s.mu.Lock()
				s.ollamaStatus.LastError = err.Error()
				s.mu.Unlock()
			} else {
				s.trace(ctx, "INFO", "Ollama genre model ready", genreModel)
			}
		}
	}

	s.mu.Lock()
	overall := "DEGRADED"
	if s.ollamaStatus.Ready && s.languageToolStatus.Ready {
		overall = "READY"
	}
	s.mu.Unlock()
	s.trace(ctx, "INFO", "Service lifecycle complete", overall)
}

func (s *serviceManager) Stop() {
	s.mu.Lock()
	ollama := s.ollamaProc
	lang := s.languageToolProc
	s.ollamaProc = nil
	s.languageToolProc = nil
	s.mu.Unlock()

	stopManagedProcess(ollama)
	stopManagedProcess(lang)
}

func (s *serviceManager) Snapshot() backend.SystemDiagnostics {
	s.mu.Lock()
	defer s.mu.Unlock()
	overall := "DEGRADED"
	if s.ollamaStatus.Ready && s.languageToolStatus.Ready {
		overall = "READY"
	} else if !s.started {
		overall = "IDLE"
	}
	copyTraces := make([]backend.ServiceTrace, len(s.traces))
	copy(copyTraces, s.traces)
	return backend.SystemDiagnostics{
		Overall:      overall,
		Initializing: s.initializing,
		Ollama:       s.ollamaStatus,
		LanguageTool: s.languageToolStatus,
		Traces:       copyTraces,
	}
}

func (s *serviceManager) SetTraceSink(sink func(backend.ServiceTrace)) {
	s.mu.Lock()
	s.traceSink = sink
	s.mu.Unlock()
}

func (s *serviceManager) updateOllama(running, ready bool, detail, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ollamaStatus = annotateServiceStatus(backend.ServiceStatus{
		Name:      "ollama",
		Running:   running,
		Ready:     ready,
		Detail:    detail,
		LastError: errMsg,
	})
}

func (s *serviceManager) updateLanguageTool(running, ready bool, detail, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.languageToolStatus = annotateServiceStatus(backend.ServiceStatus{
		Name:      "languagetool",
		Running:   running,
		Ready:     ready,
		Detail:    detail,
		LastError: errMsg,
	})
}

func (s *serviceManager) trace(ctx context.Context, level, message, detail string) {
	t := backend.ServiceTrace{Time: time.Now().Format("15:04:05.000"), Level: level, Message: message, Detail: detail}
	s.mu.Lock()
	s.traces = append(s.traces, t)
	if len(s.traces) > 400 {
		s.traces = s.traces[len(s.traces)-400:]
	}
	sink := s.traceSink
	s.mu.Unlock()
	if sink != nil {
		sink(t)
	}

	if os.Getenv("MHD_TRACE_PROGRESS") == "1" {
		fmt.Printf("%s [SERVICE] [%s] %s: %s\n", t.Time, level, message, detail)
	}

	if ctx != nil {
		runtime.EventsEmit(ctx, "analysis_progress", map[string]any{"percent": 1, "stage": "SETUP", "detail": message + ": " + detail})
		runtime.EventsEmit(ctx, "service_trace", map[string]any{
			"time":    t.Time,
			"level":   t.Level,
			"message": t.Message,
			"detail":  t.Detail,
		})
	}
}

func startOllamaServe() (*managedProcess, error) {
	ollamaBin, err := resolveBinaryPath("ollama")
	if err != nil {
		return nil, fmt.Errorf("ollama binary not found in PATH")
	}
	return startManagedProcess("ollama", ollamaBin, "serve")
}

func pullModel(model string) error {
	ollamaBin, err := resolveBinaryPath("ollama")
	if err != nil {
		return fmt.Errorf("ollama binary not found in PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, ollamaBin, "pull", model)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func startLanguageTool() (*managedProcess, error) {
	if serverBin, err := resolveBinaryPath("languagetool-server"); err == nil {
		return startManagedProcess("languagetool-server", serverBin, "--port", "8010")
	}

	if cliBin, err := resolveBinaryPath("languagetool"); err == nil {
		return startManagedProcess("languagetool", cliBin, "--http", "--port", "8010")
	}

	jar := os.Getenv("LANGUAGETOOL_JAR")
	if jar == "" {
		return nil, fmt.Errorf("languagetool binary missing and LANGUAGETOOL_JAR not set")
	}
	javaBin, err := resolveBinaryPath("java")
	if err != nil {
		return nil, fmt.Errorf("java not found while trying LANGUAGETOOL_JAR path")
	}
	return startManagedProcess("languagetool-java", javaBin, "-cp", jar, "org.languagetool.server.HTTPServer", "--port", "8010")
}

func startManagedProcess(name, bin string, args ...string) (*managedProcess, error) {
	cmd := exec.Command(bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &managedProcess{name: name, cmd: cmd}, nil
}

func stopManagedProcess(p *managedProcess) {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}

	pid := p.cmd.Process.Pid
	if pid <= 0 {
		return
	}

	_ = syscall.Kill(-pid, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_, _ = p.cmd.Process.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(3 * time.Second):
	}

	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

func isHTTPAlive(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func waitForHTTP(url string, d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if isHTTPAlive(url, 2*time.Second) {
			return
		}
		time.Sleep(900 * time.Millisecond)
	}
}

func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func annotateServiceStatus(in backend.ServiceStatus) backend.ServiceStatus {
	err := strings.ToLower(strings.TrimSpace(in.LastError))
	detail := strings.ToLower(strings.TrimSpace(in.Detail))
	combined := err + " | " + detail

	if strings.Contains(combined, "ollama binary not found") || strings.Contains(combined, "ollama not found") {
		in.Missing = true
		in.InstallCommand = "brew install ollama"
		in.InstallHint = "Ollama is missing. Install with: brew install ollama"
		return in
	}
	if strings.Contains(combined, "languagetool binary missing") ||
		strings.Contains(combined, "languagetool-server") ||
		strings.Contains(combined, "languagetool not found") {
		in.Missing = true
		in.InstallCommand = "brew install languagetool"
		in.InstallHint = "LanguageTool is missing. Install with: brew install languagetool"
		return in
	}
	if strings.Contains(combined, "java not found") {
		in.Missing = true
		in.InstallCommand = "brew install openjdk"
		in.InstallHint = "Java runtime is missing. Install with: brew install openjdk"
		return in
	}
	return in
}

func resolveBinaryPath(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	if getenv("MHD_DISABLE_SYSTEM_BIN_FALLBACK", "") == "1" {
		return "", fmt.Errorf("%s not found", name)
	}

	paths := []string{
		"/opt/homebrew/bin/" + name,
		"/usr/local/bin/" + name,
	}
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not found", name)
}

func installWithBrew(pkg string) error {
	if _, err := resolveBinaryPath("brew"); err != nil {
		return fmt.Errorf("Homebrew is not installed. Install Homebrew first from https://brew.sh")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "brew", "install", pkg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew install %s failed: %v: %s", pkg, err, strings.TrimSpace(string(out)))
	}
	return nil
}
