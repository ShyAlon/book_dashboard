package chunk

import (
	"strings"
	"testing"
)

func TestChunkingAlgorithm(t *testing.T) {
	words := make([]string, 5000)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")

	segments := SlidingWindow(text, 1500, 200)
	if len(segments) == 0 {
		t.Fatal("expected chunks to be generated")
	}

	covered := make([]bool, 5000)
	for _, s := range segments {
		if s.StartToken < 0 || s.EndToken > 5000 || s.StartToken >= s.EndToken {
			t.Fatalf("invalid segment bounds: %+v", s)
		}
		for i := s.StartToken; i < s.EndToken; i++ {
			covered[i] = true
		}
	}

	for i, ok := range covered {
		if !ok {
			t.Fatalf("data loss at token index %d", i)
		}
	}
}
