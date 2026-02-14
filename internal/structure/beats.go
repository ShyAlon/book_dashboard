package structure

type BeatWindow struct {
	Name       string
	StartRatio float64
	EndRatio   float64
}

var SaveTheCatWindows = []BeatWindow{
	{Name: "Catalyst", StartRatio: 0.10, EndRatio: 0.12},
	{Name: "Midpoint", StartRatio: 0.45, EndRatio: 0.55},
	{Name: "All is Lost", StartRatio: 0.75, EndRatio: 0.76},
}

func ChaptersInWindow(totalChapters int, startRatio, endRatio float64) (start, end int) {
	if totalChapters <= 0 {
		return 0, 0
	}
	start = int(float64(totalChapters)*startRatio) + 1
	end = int(float64(totalChapters)*endRatio) + 1
	if start < 1 {
		start = 1
	}
	if end > totalChapters {
		end = totalChapters
	}
	if start > end {
		start = end
	}
	return start, end
}
