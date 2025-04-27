package retry

import (
	"time"
)

func getDelay(o *retryOptions, tries int) time.Duration {
	backoff := o.BackOff * time.Duration(tries)
	delay := o.Delay * backoff

	if o.MaxDelay == 0 {
		return delay
	}

	if delay > o.MaxDelay {
		return o.MaxDelay
	}
	return delay
}

func Do[T any](options *retryOptions, callback func(try int) (T, error)) (T, error) {
	var tries int

	for {
		tries++

		res, err := callback(tries)
		if err == nil {
			return res, nil
		}

		switch err := err.(type) {
		case *ExitRetry:
			return res, err.Err
		case *ContinueRetry:
			if !canRetry(options, tries) {
				return res, err.Err
			}

			time.Sleep(getDelay(options, tries))
		default:
			return res, err
		}
	}
}
