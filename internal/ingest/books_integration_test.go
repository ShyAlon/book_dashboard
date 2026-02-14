package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBooksFolderPDFAndDOCX(t *testing.T) {
	booksDir := filepath.Join("..", "..", "books")
	docxPath := filepath.Join(booksDir, "The Idun Variant.docx")
	pdfPath := filepath.Join(booksDir, "The Idun Variant (no tabs).pdf")

	for _, path := range []string{docxPath, pdfPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required fixture missing: %s (%v)", path, err)
		}
		parsed, err := ParseFile(path)
		if err != nil {
			t.Fatalf("ParseFile failed for %s: %v", path, err)
		}
		if len(parsed.Text) < 200000 {
			t.Fatalf("expected substantial extracted text for %s, got %d chars", path, len(parsed.Text))
		}
	}
}
