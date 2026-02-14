package forensics

import "testing"

func TestContradictionEngine(t *testing.T) {
	input := []ChapterProfile{
		{
			Chapter: 1,
			Name:    "John",
			Attributes: map[string]string{
				"dead": "false",
			},
		},
		{
			Chapter: 5,
			Name:    "John",
			Attributes: map[string]string{
				"dead": "true",
			},
		},
	}

	contradictions := DetectContradictions(input)
	if len(contradictions) == 0 {
		t.Fatal("expected contradiction to be detected")
	}

	if contradictions[0].Severity != "HIGH" {
		t.Fatalf("expected HIGH severity, got %s", contradictions[0].Severity)
	}
}
