package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"book_dashboard/internal/ingest"
)

func TestAnalyzeBooksDOCX_RichAnalysis(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	booksDir := filepath.Join("..", "books")
	docxPath := filepath.Join(booksDir, "The Idun Variant.docx")
	if _, err := os.Stat(docxPath); err != nil {
		t.Fatalf("required fixture missing: %s (%v)", docxPath, err)
	}

	docxApp := NewApp()
	docxData := docxApp.AnalyzeFile(docxPath)
	assertCommonAnalysisOutput(t, docxPath, docxData.ProjectLocation)
	if docxData.WordCount < 30000 {
		t.Fatalf("expected large manuscript word count for %s, got %d", docxPath, docxData.WordCount)
	}
	if docxData.RunStats.SegmentCount < 10 {
		t.Fatalf("expected multiple segments for %s, got %d", docxPath, docxData.RunStats.SegmentCount)
	}
	docxReportPath := filepath.Join(docxData.ProjectLocation, "report.json")
	assertRichReportJSON(t, docxReportPath)
	t.Logf(
		"analysis summary | file=%s | words=%d | chapters=%d | segments=%d | score=%d | report=%s",
		docxPath,
		docxData.WordCount,
		docxData.ChapterCount,
		docxData.RunStats.SegmentCount,
		docxData.MHDScore,
		docxReportPath,
	)
	maybeCopyArtifacts(t, docxPath, docxReportPath)
}

func TestAnalyzeBooksPDF_ChapterizationOnly(t *testing.T) {
	booksDir := filepath.Join("..", "books")
	pdfPath := filepath.Join(booksDir, "The Idun Variant (no tabs).pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("required fixture missing: %s (%v)", pdfPath, err)
	}

	parsedPDF, err := ingest.ParseFile(pdfPath)
	if err != nil {
		t.Fatalf("parse pdf fixture failed: %v", err)
	}
	pdfWordCount := len(strings.Fields(parsedPDF.Text))
	if pdfWordCount < 30000 {
		t.Fatalf("expected large manuscript word count for %s, got %d", pdfPath, pdfWordCount)
	}
	pdfChapterCount := estimateChapterHeadings(parsedPDF.Text)
	if pdfChapterCount < 10 {
		t.Fatalf("expected PDF chapter heading detection to find enough chapters for %s, got %d", pdfPath, pdfChapterCount)
	}
	t.Logf(
		"pdf ingest summary | file=%s | words=%d | chapters=%d",
		pdfPath,
		pdfWordCount,
		pdfChapterCount,
	)
}

func TestAnalyzeBooksDOCX_AISlopDetection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	booksDir := filepath.Join("..", "books")
	docxPath := filepath.Join(booksDir, "THE IDUN PROTOCOL (AI Slop).docx")
	if _, err := os.Stat(docxPath); err != nil {
		t.Fatalf("required fixture missing: %s (%v)", docxPath, err)
	}

	app := NewApp()
	data := app.AnalyzeFile(docxPath)
	if data.AIReport.PAIDoc == nil || data.AIReport.AICoverageEst == nil || data.AIReport.PAIMax == nil {
		t.Fatalf("expected ai report probabilities for %s, got %+v", docxPath, data.AIReport)
	}
	if *data.AIReport.PAIDoc < 0.65 {
		t.Fatalf("expected high document AI probability for %s, got %.3f", docxPath, *data.AIReport.PAIDoc)
	}
	if *data.AIReport.PAIMax < 0.85 {
		t.Fatalf(
			"expected high max window AI signal for %s, got p_ai_max=%.3f",
			docxPath,
			*data.AIReport.PAIMax,
		)
	}
	if *data.AIReport.AICoverageEst < 0.10 {
		t.Fatalf("expected non-trivial AI coverage for %s, got %.3f", docxPath, *data.AIReport.AICoverageEst)
	}
	if data.MHDScore >= 80 {
		t.Fatalf("expected AI-likelihood score penalty for %s, got MHD score %d", docxPath, data.MHDScore)
	}
	flagText := strings.ToLower(strings.Join(data.AIReport.Flags, " | "))
	if !strings.Contains(flagText, "ai_chunk_detected") && !strings.Contains(flagText, "possible_stitching") {
		t.Fatalf("expected AI duplication/stitching flags for %s, got %v", docxPath, data.AIReport.Flags)
	}
}

var chapterHeadingPattern = regexp.MustCompile(`(?im)^\s*(chapter|ch\.?)\s+([0-9ivxlcdm]+)\b`)

func estimateChapterHeadings(text string) int {
	return len(chapterHeadingPattern.FindAllString(text, -1))
}

func assertCommonAnalysisOutput(t *testing.T, path, projectLocation string) {
	t.Helper()

	if projectLocation == "" {
		t.Fatalf("expected project location for %s", path)
	}
	if _, err := os.Stat(filepath.Join(projectLocation, "report.json")); err != nil {
		t.Fatalf("report.json not persisted for %s: %v", path, err)
	}
}

func safeName(path string) string {
	base := filepath.Base(path)
	base = strings.ReplaceAll(base, " ", "_")
	base = strings.ReplaceAll(base, "(", "")
	base = strings.ReplaceAll(base, ")", "")
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func maybeCopyArtifacts(t *testing.T, path, reportPath string) {
	t.Helper()
	if os.Getenv("KEEP_E2E_ARTIFACTS") != "1" {
		return
	}

	dstDir := filepath.Join(".", "test-artifacts", "e2e", safeName(path))
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("create artifacts dir: %v", err)
	}
	if err := copyFile(reportPath, filepath.Join(dstDir, "report.json")); err != nil {
		t.Fatalf("copy report artifact: %v", err)
	}
	t.Logf("artifacts saved to %s", dstDir)
}

func assertRichReportJSON(t *testing.T, reportPath string) {
	t.Helper()
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}

	analysis, ok := report["analysis"].(map[string]any)
	if !ok {
		t.Fatalf("report missing analysis payload")
	}

	language, ok := analysis["language"].(map[string]any)
	if !ok {
		t.Fatalf("analysis missing language section")
	}
	if _, ok := language["spellingScore"]; !ok {
		t.Fatalf("language section missing spellingScore")
	}
	if _, ok := language["grammarScore"]; !ok {
		t.Fatalf("language section missing grammarScore")
	}

	if scores, ok := analysis["genre_scores"].([]any); !ok || len(scores) == 0 {
		t.Fatalf("analysis missing genre_scores")
	}
	if provider, _ := analysis["genre_provider"].(string); !strings.HasPrefix(provider, "ollama:") {
		reason, _ := analysis["genre_reasoning"].(string)
		t.Fatalf("expected genre_provider from ollama, got %q (reasoning=%q)", provider, reason)
	}
	if reasoning, _ := analysis["genre_reasoning"].(string); strings.TrimSpace(reasoning) == "" {
		t.Fatalf("analysis missing genre_reasoning")
	}
	if metrics, ok := analysis["chapter_metrics"].([]any); !ok || len(metrics) == 0 {
		t.Fatalf("analysis missing chapter_metrics")
	} else {
		for i, item := range metrics {
			m, ok := item.(map[string]any)
			if !ok {
				t.Fatalf("chapter_metrics[%d] invalid", i)
			}
			if provider, _ := m["genreProvider"].(string); !strings.HasPrefix(provider, "ollama:") {
				reasoning, _ := m["genreReasoning"].(string)
				t.Fatalf("expected chapter_metrics[%d].genreProvider from ollama, got %q (reasoning=%q)", i, provider, reasoning)
			}
			if reasoning, _ := m["genreReasoning"].(string); strings.TrimSpace(reasoning) == "" {
				t.Fatalf("chapter_metrics[%d] missing genreReasoning", i)
			}
		}
	}
	plot, ok := analysis["plot_structure"].(map[string]any)
	if !ok {
		t.Fatalf("analysis missing plot_structure")
	}
	if provider, _ := plot["provider"].(string); strings.TrimSpace(provider) == "" {
		t.Fatalf("plot_structure missing provider")
	}
	if probs, ok := plot["probabilities"].([]any); !ok || len(probs) < 3 {
		t.Fatalf("plot_structure missing probabilities")
	}
	if selected, _ := plot["selectedStructure"].(string); strings.TrimSpace(selected) == "" {
		t.Fatalf("plot_structure missing selectedStructure")
	}
	if reasoning, _ := plot["reasoning"].(string); strings.TrimSpace(reasoning) == "" {
		t.Fatalf("plot_structure missing reasoning")
	}
	if beats, ok := analysis["beats"].([]any); !ok || len(beats) == 0 {
		t.Fatalf("analysis missing beats")
	}
	if aiReport, ok := analysis["ai_report"].(map[string]any); !ok {
		t.Fatalf("analysis missing ai_report")
	} else {
		if _, ok := aiReport["p_ai_doc"]; !ok {
			t.Fatalf("analysis.ai_report missing p_ai_doc")
		}
		if windows, ok := aiReport["windows"].([]any); !ok || len(windows) == 0 {
			t.Fatalf("analysis.ai_report missing windows")
		}
	}

	if chapterSummaries, ok := analysis["chapter_summaries"].([]any); !ok || len(chapterSummaries) == 0 {
		t.Fatalf("analysis missing chapter_summaries")
	} else {
		first, ok := chapterSummaries[0].(map[string]any)
		if !ok {
			t.Fatalf("chapter_summaries[0] invalid")
		}
		if events, ok := first["events"].([]any); !ok || len(events) == 0 {
			t.Fatalf("chapter_summaries[0] missing events")
		}
	}

	runStats, ok := analysis["run_stats"].(map[string]any)
	if !ok {
		t.Fatalf("analysis missing run_stats")
	}
	if status, _ := runStats["status"].(string); status != "DONE" {
		t.Fatalf("expected run_stats.status DONE, got %q", status)
	}
}
