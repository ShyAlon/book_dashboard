package slop

import (
	"strings"
	"testing"
)

func TestAnalyzeFlagsAIGenerationArtifacts(t *testing.T) {
	repeatedBlock := strings.TrimSpace(`
The metallic tang of fear sat on his tongue as he crossed the sterile hallway and listened to the hollow scrape of his own shoes.
Every polished surface reflected the same grave certainty: the process would continue, the directive would remain, and compliance would be measured.
He tried to remember the warmth of ordinary life, but the room answered with efficiency, optimization, and a perfectly obedient silence.
`)
	text := strings.Join([]string{
		"Chapter 1: The Last Breath",
		repeatedBlock,
		"",
		"THE IDUN PROTOCOL (Elaborated Version)",
		repeatedBlock,
		"",
		"Chapter 3: The Empty Room",
		repeatedBlock,
		"",
		"Chapter 4: The Last Breath",
		repeatedBlock,
	}, "\n\n")

	report := Analyze(text)
	if report.RepeatedBlockCount == 0 {
		t.Fatalf("expected repeated blocks to be detected")
	}
	if report.MaxBlockRepeat < 3 {
		t.Fatalf("expected max block repeat >= 3, got %d", report.MaxBlockRepeat)
	}
	if report.VerbatimDuplicationCoverage < 0.20 {
		t.Fatalf("expected high duplication coverage, got %.3f", report.VerbatimDuplicationCoverage)
	}
	if report.AISuspicionScore < 45 {
		t.Fatalf("expected ai suspicion score >= 45, got %d", report.AISuspicionScore)
	}
	if !report.LikelyAIGenerated {
		t.Fatalf("expected likely ai generated to be true")
	}
	joinedFlags := strings.ToLower(strings.Join(report.Flags, " | "))
	if !strings.Contains(joinedFlags, "verbatim") {
		t.Fatalf("expected verbatim repetition flag, got %+v", report.Flags)
	}
}

func TestAnalyzeDoesNotOverFlagNormalDraft(t *testing.T) {
	parts := []string{
		"Chapter 1: Morning",
		"He walked to the station, bought coffee, and missed his train by one minute.",
		"The delay made him call his sister, and they argued briefly about their father.",
		"By noon he had made up his mind to visit home.",
		"",
		"Chapter 2: Evening",
		"Rain started around dinner and the streets filled with umbrellas.",
		"He cooked soup, answered two emails, and read old notes before sleeping.",
	}
	report := Analyze(strings.Join(parts, "\n\n"))
	if report.LikelyAIGenerated {
		t.Fatalf("expected normal draft not to be marked as likely ai generated (score=%d flags=%v)", report.AISuspicionScore, report.Flags)
	}
}
