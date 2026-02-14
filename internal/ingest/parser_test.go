package ingest

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDOCX(t *testing.T) {
	raw := buildDOCX(t, `<w:document><w:body><w:p><w:r><w:t>Chapter 1</w:t></w:r></w:p><w:p><w:r><w:t>Hello world.</w:t></w:r></w:p></w:body></w:document>`)
	got, err := parseDOCX(raw)
	if err != nil {
		t.Fatalf("parseDOCX failed: %v", err)
	}
	if got == "" {
		t.Fatal("expected extracted text")
	}
}

func TestParseFileUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected unsupported file type error")
	}
}

func buildDOCX(t *testing.T, bodyXML string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>` + bodyXML
	if _, err := f.Write([]byte(xml)); err != nil {
		t.Fatalf("write xml: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return b.Bytes()
}
