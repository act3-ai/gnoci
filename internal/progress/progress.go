package progress

import (
	"context"
	"time"
)

// Inspired by https://github.com/machinebox/progress/blob/master/progress.go.

// Evaluator facilitates progress monitoring.
type Evaluator interface {
	// Progress returns a total, a delta since it's last call, and any error
	// encountered since the last call to Progress.
	Progress() (int, int, error)
}

// Progress is an message reporting a cumulative total and change since the last
// Progress message.
type Progress struct {
	// Total is the cumulative total.
	Total int
	// Delta is the difference between Total and the previous message's Total.
	Delta int
}

// Ticker holds a channel that delivers "ticks" of [Progress] at intervals.
type Ticker struct {
	C <-chan Progress
}

// NewTicker returns a [Ticker] reporting an [Evaluator]'s [Progress] on an interval.
func NewTicker(ctx context.Context, eval Evaluator, d time.Duration) *Ticker {
	ch := make(chan Progress)
	t := time.NewTicker(d)

	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
			case <-t.C:
				total, delta, err := eval.Progress()
				p := Progress{
					Total: total,
					Delta: delta,
				}

				ch <- p
				if err != nil { // io.EOF, or other issues
					return
				}
			}
		}
	}()

	return &Ticker{C: ch}
}
