package retry

import "time"

type retryOptions struct {
	Tries    int
	Delay    time.Duration
	MaxDelay time.Duration
	BackOff  time.Duration
}

func NewOptions(options ...func(*retryOptions)) *retryOptions {
	o := &retryOptions{
		Tries:    -1,
		Delay:    1 * time.Second,
		MaxDelay: 0,
		BackOff:  1,
	}

	for _, option := range options {
		option(o)
	}

	return o
}

func Tries(tries int) func(*retryOptions) {
	return func(o *retryOptions) {
		o.Tries = tries
	}
}

func Delay(delay time.Duration) func(*retryOptions) {
	return func(o *retryOptions) {
		o.Delay = delay
	}
}

func MaxDelay(maxDelay time.Duration) func(*retryOptions) {
	return func(o *retryOptions) {
		o.MaxDelay = maxDelay
	}
}

func BackOff(backOff time.Duration) func(*retryOptions) {
	return func(o *retryOptions) {
		o.BackOff = backOff
	}
}

func canRetry(o *retryOptions, tries int) bool {
	if tries == -1 || tries < o.Tries {
		return true
	}
	return false
}
