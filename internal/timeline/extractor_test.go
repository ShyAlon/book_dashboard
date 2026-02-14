package timeline

import "testing"

func TestEventsFromText(t *testing.T) {
	text := "In 1999 we met. The next day everything changed. Last night he disappeared."
	events := EventsFromText(text, 10)
	if len(events) < 2 {
		t.Fatalf("expected timeline events, got %d", len(events))
	}
	if events[0].TimeMarker == "" || events[0].Event == "" {
		t.Fatal("expected populated event")
	}
}
