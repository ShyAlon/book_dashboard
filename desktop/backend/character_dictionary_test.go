package backend

import "testing"

func TestNamesInTextFiltersDialogueFillersButKeepsCharacterNames(t *testing.T) {
	text := `
Well, that is surprising.
Maybe we should leave.
Not now.
What happened here?
Dawn said she was ready.
I met Dawn at the station.
`

	got := namesInText(text)
	gotSet := map[string]struct{}{}
	for _, n := range got {
		gotSet[n] = struct{}{}
	}

	if _, ok := gotSet["Dawn"]; !ok {
		t.Fatalf("expected Dawn to be kept as character candidate, got=%v", got)
	}

	for _, bad := range []string{"Well", "Maybe", "Not", "What"} {
		if _, ok := gotSet[bad]; ok {
			t.Fatalf("expected %q to be filtered out, got=%v", bad, got)
		}
	}
}
