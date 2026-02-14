package backend

func clamp100(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const DefaultDemoText = `Chapter 1
In 1999, John returned to the town after years away. The next day he found a letter on his doorstep.
He said the same line every morning. He said the same line every morning. Last night, someone erased the camera feed.

Chapter 2
John's eyes were blue when he entered the station. The detective said he was alive.

Chapter 12
John's eyes were brown after the accident. The witness said John was dead.`
