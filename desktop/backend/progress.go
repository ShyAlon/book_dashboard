package backend

type ProgressFn func(percent int, stage, detail string)

func progress(on ProgressFn, percent int, stage, detail string) {
	if on == nil {
		return
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	on(percent, stage, detail)
}
