package offline

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"book_dashboard/internal/chunk"
	"book_dashboard/internal/forensics"
	"book_dashboard/internal/slop"
)

type failTransport struct{}

func (f failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network disabled for offline test")
}

func TestOfflineMode(t *testing.T) {
	original := http.DefaultTransport
	http.DefaultTransport = failTransport{}
	t.Cleanup(func() { http.DefaultTransport = original })

	text := strings.Repeat("This is a sentence. ", 500)
	segments := chunk.SlidingWindow(text, 1500, 200)
	if len(segments) == 0 {
		t.Fatal("expected chunking to work offline")
	}

	profiles := []forensics.ChapterProfile{
		{Chapter: 1, Name: "John", Attributes: map[string]string{"dead": "false"}},
		{Chapter: 2, Name: "John", Attributes: map[string]string{"dead": "true"}},
	}
	issues := forensics.DetectContradictions(profiles)
	if len(issues) == 0 {
		t.Fatal("expected contradiction detection to work offline")
	}

	report := slop.Analyze(text)
	if report.MeanSentenceLength == 0 {
		t.Fatal("expected slop analysis to work offline")
	}
}
