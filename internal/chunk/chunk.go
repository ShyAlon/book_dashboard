package chunk

import "strings"

type Segment struct {
	Index      int
	StartToken int
	EndToken   int
	Text       string
}

func SlidingWindow(text string, segmentTokens, overlapTokens int) []Segment {
	if segmentTokens <= 0 {
		return nil
	}
	if overlapTokens < 0 {
		overlapTokens = 0
	}
	if overlapTokens >= segmentTokens {
		overlapTokens = segmentTokens - 1
	}

	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return nil
	}

	step := segmentTokens - overlapTokens
	segments := make([]Segment, 0, (len(tokens)/step)+1)
	for start := 0; start < len(tokens); start += step {
		end := start + segmentTokens
		if end > len(tokens) {
			end = len(tokens)
		}
		segments = append(segments, Segment{
			Index:      len(segments),
			StartToken: start,
			EndToken:   end,
			Text:       strings.Join(tokens[start:end], " "),
		})
		if end == len(tokens) {
			break
		}
	}

	return segments
}
