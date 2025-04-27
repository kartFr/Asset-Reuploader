package taskqueue

import (
	"container/list"
	"sync"
	"time"
)

type TaskResult[R any] struct {
	Result R
	Error  error
}

type task[R any] struct {
	Func func() (R, error)
	Chan chan TaskResult[R]
}

type Queue[R any] struct {
	Limiter *fixedWindow

	interval           time.Duration
	isSchedulerRunning bool
	mutex              sync.Mutex
	tasks              *list.List
}

func New[R any](window time.Duration, limit int) *Queue[R] {
	return &Queue[R]{
		Limiter: newFixedWindow(window, limit),

		interval: window / time.Duration(limit),
		tasks:    list.New(),
	}
}

func (q *Queue[R]) QueueTask(f func() (R, error)) chan TaskResult[R] {
	c := make(chan TaskResult[R])

	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.tasks.PushBack(task[R]{
		Func: f,
		Chan: c,
	})

	if !q.isSchedulerRunning {
		q.isSchedulerRunning = true
		go q.scheduler()
	}

	return c
}

func (q *Queue[R]) scheduler() {
	for {
		q.mutex.Lock()
		if q.tasks.Len() == 0 {
			q.isSchedulerRunning = false
			q.mutex.Unlock()
			return
		}

		e := q.tasks.Front()
		t := e.Value.(task[R])

		q.tasks.Remove(e)
		q.mutex.Unlock()

		q.Limiter.Wait()
		go func() {
			res, err := t.Func()
			t.Chan <- TaskResult[R]{
				Result: res,
				Error:  err,
			}
		}()

		time.Sleep(q.interval)
	}
}
