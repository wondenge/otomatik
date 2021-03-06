package otomatik

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

var jobmanager = &jobManager{maxConcurrentJobs: 1000}

type jobManager struct {
	mu                sync.Mutex
	maxConcurrentJobs int
	activeWorkers     int
	queue             []namedJob
	names             map[string]struct{}
}

type namedJob struct {
	name string
	job  func() error
}

// Submit enqueues the given job with the given name.
// If name is non-empty and a job with the same name is already enqueued or running, this is a no-op.
// If name is empty, no duplicate prevention will occur.
// The job manager will then run this job as soon as it is able.
func (jobmanager *jobManager) Submit(name string, job func() error) {
	jobmanager.mu.Lock()
	defer jobmanager.mu.Unlock()
	if jobmanager.names == nil {
		jobmanager.names = make(map[string]struct{})
	}
	if name != "" {
		// prevent duplicate jobs
		if _, ok := jobmanager.names[name]; ok {
			return
		}
		jobmanager.names[name] = struct{}{}
	}
	jobmanager.queue = append(jobmanager.queue, namedJob{name, job})
	if jobmanager.activeWorkers < jobmanager.maxConcurrentJobs {
		jobmanager.activeWorkers++
		go jobmanager.worker()
	}
}

func (jobmanager *jobManager) worker() {
	for {
		jobmanager.mu.Lock()
		if len(jobmanager.queue) == 0 {
			jobmanager.activeWorkers--
			jobmanager.mu.Unlock()
			return
		}
		next := jobmanager.queue[0]
		jobmanager.queue = jobmanager.queue[1:]
		jobmanager.mu.Unlock()
		if err := next.job(); err != nil {
			log.Printf("[ERROR] %v", err)
		}
		if next.name != "" {
			jobmanager.mu.Lock()
			delete(jobmanager.names, next.name)
			jobmanager.mu.Unlock()
		}
	}
}

func doWithRetry(ctx context.Context, f func(context.Context) error) error {
	var attempts int
	ctx = context.WithValue(ctx, AttemptsCtxKey, &attempts)

	// the initial intervalIndex is -1, signaling that we should not wait for the first attempt
	start, intervalIndex := time.Now(), -1
	var err error

	for time.Since(start) < maxRetryDuration {
		var wait time.Duration
		if intervalIndex >= 0 {
			wait = retryIntervals[intervalIndex]
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return context.Canceled
		case <-timer.C:
			err = f(ctx)
			attempts++
			if err == nil || errors.Is(err, context.Canceled) {
				return err
			}
			var errNoRetry ErrNoRetry
			if errors.As(err, &errNoRetry) {
				return err
			}
			if intervalIndex < len(retryIntervals)-1 {
				intervalIndex++
			}
			if time.Since(start) < maxRetryDuration {
				log.Printf("[ERROR] attempt %d: %v - retrying in %s (%s/%s elapsed)...",
					attempts, err, retryIntervals[intervalIndex], time.Since(start), maxRetryDuration)
			} else {
				log.Printf("[ERROR] final attempt: %v - giving up (%s/%s elapsed)...",
					err, time.Since(start), maxRetryDuration)
				return nil
			}
		}
	}
	return err
}

// ErrNoRetry is an error type which signals to stop retries early.
type ErrNoRetry struct{ Err error }

// Unwrap makes it so that e wraps e.Err.
func (e ErrNoRetry) Unwrap() error { return e.Err }
func (e ErrNoRetry) Error() string { return e.Err.Error() }

type retryStateCtxKey struct{}

// AttemptsCtxKey is the context key for the value that holds the attempt counter.
// The value counts how many times the operation has been attempted.
// A value of 0 means first attempt.
var AttemptsCtxKey retryStateCtxKey

// retryIntervals are based on the idea of exponential backoff, but weighed a little more heavily to the front.
// We figure that intermittent errors would be resolved after the first retry,
// but any errors after that would probably require at least a few minutes to clear up:
// either for DNS to propagate, for the administrator to fix their DNS or network properties,
// or some other external factor needs to change.
// We chose intervals that we think will be most useful without introducing unnecessary delay.
// The last interval in this list will be used until the time of maxRetryDuration has elapsed.
var retryIntervals = []time.Duration{
	1 * time.Minute,
	2 * time.Minute,
	2 * time.Minute,
	5 * time.Minute, // elapsed: 10 min
	10 * time.Minute,
	20 * time.Minute,
	20 * time.Minute, // elapsed: 1 hr
	30 * time.Minute,
	30 * time.Minute, // elapsed: 2 hrs
	1 * time.Hour,
	3 * time.Hour, // elapsed: 6 hr
	6 * time.Hour, // for up to maxRetryDuration
}

// maxRetryDuration is the maximum duration to try doing retries using the above intervals.
const maxRetryDuration = 24 * time.Hour * 30
