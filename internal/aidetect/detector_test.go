package aidetect

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
)

type stubLT struct {
	score float64
	err   error
}

func (s stubLT) ScoreWindow(context.Context, string) (float64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.score, nil
}

type stubLM struct {
	score float64
	err   error
}

func (s stubLM) ScoreWindow(context.Context, string) (float64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.score, nil
}

func TestSegmentWindowsBoundaries(t *testing.T) {
	words := make([]string, 1200)
	for i := range words {
		words[i] = "word"
	}
	windows := segmentWindows(words, 900, 450)
	if len(windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(windows))
	}
	if windows[0].Start != 0 || windows[0].End != 900 {
		t.Fatalf("unexpected first window: %+v", windows[0])
	}
	if windows[1].Start != 450 || windows[1].End != 1200 {
		t.Fatalf("unexpected second window: %+v", windows[1])
	}
}

func TestShingleSetDeterministic(t *testing.T) {
	words := strings.Fields("one two three four five six seven eight nine ten eleven twelve")
	a := shingleSet(words, 10)
	b := shingleSet(words, 10)
	if len(a) != len(b) {
		t.Fatalf("expected equal shingle counts, got %d vs %d", len(a), len(b))
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			t.Fatalf("shingle key missing in second run")
		}
	}
}

func TestDuplicationDetectionExactParagraphs(t *testing.T) {
	paragraph := strings.TrimSpace(strings.Repeat("the sterile corridor hummed with certainty and fear ", 48))
	text := strings.Join([]string{paragraph, "", "chapter break", "", paragraph}, "\n\n")
	cfg := DefaultConfig()
	cfg.EnableLanguageTool = false
	report := Analyze(Input{DocumentID: "dup-doc", Text: text, Language: "en"}, cfg, nil, nil, nil)
	if report.PAIDoc == nil {
		t.Fatalf("expected document probability")
	}
	if *report.PAIDoc < 0.65 {
		t.Fatalf("expected high p_ai_doc for exact duplication, got %.3f", *report.PAIDoc)
	}
	if len(report.Flags) == 0 {
		t.Fatalf("expected flags for duplication")
	}
	foundLongDup := false
	for _, w := range report.Windows {
		for _, e := range w.TopEvidence {
			if e.Type == "duplication" && strings.Contains(strings.ToLower(e.Summary), "long duplicate span") {
				foundLongDup = true
			}
		}
	}
	if !foundLongDup {
		t.Fatalf("expected long duplicate span evidence")
	}
}

func TestDuplicationNoFalsePositiveOnSmallRepeats(t *testing.T) {
	smallRepeat := "the room was cold and bright"
	parts := make([]string, 0, 2200)
	for i := 0; i < 900; i++ {
		parts = append(parts, "Record "+strconv.Itoa(i)+" notes a distinct weather pattern near district "+strconv.Itoa(i%27)+".")
		parts = append(parts, "Witness "+strconv.Itoa((i*7)%503)+" described a different sequence of events around marker "+strconv.Itoa(i%113)+".")
		parts = append(parts, "Archive entry "+strconv.Itoa(i*i%997)+" references materials stored in bay "+strconv.Itoa((i*11)%71)+".")
		if i%40 == 0 {
			parts = append(parts, smallRepeat)
		}
	}
	text := strings.Join(parts, " ")
	cfg := DefaultConfig()
	cfg.EnableLanguageTool = false
	report := Analyze(Input{DocumentID: "small-repeat", Text: text, Language: "en"}, cfg, nil, nil, nil)
	if report.PAIDoc == nil {
		t.Fatalf("expected p_ai_doc")
	}
	for _, w := range report.Windows {
		for _, e := range w.TopEvidence {
			if strings.Contains(strings.ToLower(e.Summary), "long duplicate span") {
				t.Fatalf("did not expect long duplicate span override")
			}
		}
	}
}

func TestAnalyzeLanguageToolFailureStillProducesOutput(t *testing.T) {
	text := strings.TrimSpace(strings.Repeat("The hall remained silent while every clock refused to move. ", 1200))
	cfg := DefaultConfig()
	cfg.EnableLanguageTool = true
	report := Analyze(Input{DocumentID: "lt-fail", Text: text, Language: "en"}, cfg, stubLT{err: errors.New("connection refused")}, nil, nil)
	if report.PAIDoc == nil || report.AICoverageEst == nil || report.PAIMax == nil {
		t.Fatalf("expected aggregated output despite LT failure")
	}
	if len(report.Errors) == 0 {
		t.Fatalf("expected errors for LT failure")
	}
	foundLT := false
	for _, err := range report.Errors {
		if err.Stage == "language_tool_run" {
			foundLT = true
		}
	}
	if !foundLT {
		t.Fatalf("expected language_tool_run error entry")
	}
}

func TestAnalyzeLMFailureRedistributesWeights(t *testing.T) {
	w := signalWeights(false)
	if w.Duplication != 0.50 || w.StyleUniform != 0.30 || w.PolishCliche != 0.15 || w.LMSmoothness != 0.0 {
		t.Fatalf("unexpected redistributed weights: %+v", w)
	}

	text := strings.TrimSpace(strings.Repeat("She archived the notes and closed the ledger before sunrise. ", 1500))
	cfg := DefaultConfig()
	cfg.EnableLanguageTool = false
	cfg.EnableLMSmoothness = true
	report := Analyze(Input{DocumentID: "lm-fail", Text: text, Language: "en"}, cfg, nil, stubLM{err: errors.New("deadline exceeded")}, nil)
	if report.PAIDoc == nil {
		t.Fatalf("expected p_ai_doc despite LM failure")
	}
	foundLM := false
	for _, err := range report.Errors {
		if err.Stage == "lm_scoring_run" {
			foundLM = true
		}
	}
	if !foundLM {
		t.Fatalf("expected lm_scoring_run error entry")
	}
	for _, wr := range report.Windows {
		if wr.Signals.LMSmoothness.Score != nil {
			t.Fatalf("expected lm_smoothness score to be nil when LM fails")
		}
	}
}

func TestDocumentProbabilityDoesNotSaturateOnModerateSignals(t *testing.T) {
	text := strings.TrimSpace(strings.Repeat("The committee reviewed the report and scheduled a follow up meeting for next week. ", 5000))
	cfg := DefaultConfig()
	cfg.EnableLanguageTool = false
	report := Analyze(Input{DocumentID: "moderate-doc", Text: text, Language: "en"}, cfg, nil, nil, nil)
	if report.PAIDoc == nil || report.PAIMax == nil {
		t.Fatalf("expected probabilities")
	}
	if *report.PAIDoc > 0.80 && *report.PAIMax < 0.70 {
		t.Fatalf("unexpected doc saturation: p_ai_doc=%.3f p_ai_max=%.3f", *report.PAIDoc, *report.PAIMax)
	}
}
