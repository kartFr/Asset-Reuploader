package taskqueue

import (
	"sync"
	"time"
)

type fixedWindow struct {
	requests int
	window   time.Duration
	limit    int
	start    time.Time
	mu       sync.Mutex
}

func newFixedWindow(window time.Duration, limit int) *fixedWindow {
	return &fixedWindow{
		window: window,
		limit:  limit,
		start:  time.Now(),
	}
}

func (w *fixedWindow) timeRemaining(t time.Time) time.Duration {
	reset := w.start.Add(w.window)
	return reset.Sub(t)
}

func (w *fixedWindow) Increment() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timeRemaining(time.Now()) < 0 {
		w.start = time.Now()
		w.requests = max(w.requests-w.limit, 0)
	}

	if w.requests >= w.limit {
		return false
	}

	w.requests++
	return true
}

func (w *fixedWindow) Decrement() {
	w.mu.Lock()
	w.requests--
	w.mu.Unlock()
}

func (w *fixedWindow) Wait() {
	if w.Increment() {
		return
	}

	w.mu.Lock()

	w.requests++
	nextWindow := (w.requests-1)/w.limit + 1
	nextWindowTime := w.start.Add(w.window * time.Duration(nextWindow))

	w.mu.Unlock()

	time.Sleep(w.timeRemaining(nextWindowTime))
	w.Decrement()
}
