package pipeline

import (
	"runtime"
	"sync"

	"book_dashboard/internal/chunk"
)

type Analyzer func(seg chunk.Segment) error

func AnalyzeSegments(segments []chunk.Segment, workers int, fn Analyzer) []error {
	if len(segments) == 0 || fn == nil {
		return nil
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 1 {
			workers = 1
		}
	}

	jobs := make(chan chunk.Segment)
	errs := make(chan error, len(segments))
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for seg := range jobs {
				if err := fn(seg); err != nil {
					errs <- err
				}
			}
		}()
	}

	for _, seg := range segments {
		jobs <- seg
	}
	close(jobs)
	wg.Wait()
	close(errs)

	out := make([]error, 0, len(errs))
	for err := range errs {
		out = append(out, err)
	}
	return out
}
