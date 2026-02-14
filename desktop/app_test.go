package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeFilePersistsReport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	docxPath := filepath.Join(t.TempDir(), "Sample.docx")
	if err := os.WriteFile(docxPath, buildDOCX(t), 0o644); err != nil {
		t.Fatalf("write docx: %v", err)
	}

	app := NewApp()
	data := app.AnalyzeFile(docxPath)
	if data.WordCount == 0 {
		t.Fatal("expected non-zero word count from parsed docx")
	}
	if data.ProjectLocation == "" {
		t.Fatal("expected project location")
	}

	reportPath := filepath.Join(data.ProjectLocation, "report.json")
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report.json at %s: %v", reportPath, err)
	}
	if data.ChapterCount == 0 {
		t.Fatal("expected chapter detection to run")
	}
	if len(data.ChapterMetrics) == 0 {
		t.Fatal("expected chapter metrics to be populated")
	}
}

func buildDOCX(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create docx: %v", err)
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?><w:document><w:body><w:p><w:r><w:t>Chapter 1</w:t></w:r></w:p><w:p><w:r><w:t>John arrives in 1999.</w:t></w:r></w:p></w:body></w:document>`
	if _, err := f.Write([]byte(xml)); err != nil {
		t.Fatalf("write xml: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return b.Bytes()
}
