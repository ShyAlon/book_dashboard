package db

import (
	"path/filepath"
	"testing"

	"book_dashboard/internal/forensics"
)

func TestPersistContradictions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "analysis.db")
	input := []forensics.Contradiction{
		{
			EntityName:  "john",
			ChapterA:    1,
			ChapterB:    12,
			Description: "eyes changed from blue to brown",
			Severity:    "MED",
		},
		{
			EntityName:  "john",
			ChapterA:    2,
			ChapterB:    13,
			Description: "dead changed from false to true",
			Severity:    "HIGH",
		},
	}

	if err := PersistContradictions(dbPath, input); err != nil {
		t.Fatalf("persist contradictions: %v", err)
	}

	entities, err := CountRows(dbPath, "entities")
	if err != nil {
		t.Fatalf("count entities: %v", err)
	}
	if entities != 1 {
		t.Fatalf("expected 1 entity, got %d", entities)
	}

	contradictions, err := CountRows(dbPath, "contradictions")
	if err != nil {
		t.Fatalf("count contradictions: %v", err)
	}
	if contradictions != 2 {
		t.Fatalf("expected 2 contradictions, got %d", contradictions)
	}
}
