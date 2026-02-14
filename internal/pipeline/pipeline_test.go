package pipeline

import (
	"errors"
	"sync/atomic"
	"testing"

	"book_dashboard/internal/chunk"
)

func TestAnalyzeSegments(t *testing.T) {
	segs := []chunk.Segment{
		{Index: 0, Text: "a"},
		{Index: 1, Text: "b"},
		{Index: 2, Text: "c"},
	}

	var called int32
	errs := AnalyzeSegments(segs, 2, func(seg chunk.Segment) error {
		atomic.AddInt32(&called, 1)
		if seg.Index == 1 {
			return errors.New("test error")
		}
		return nil
	})

	if called != int32(len(segs)) {
		t.Fatalf("expected %d calls, got %d", len(segs), called)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}
