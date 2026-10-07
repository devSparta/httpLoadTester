package runner

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

type Config struct {
	URL         string
	Method      string
	Requests    int
	Concurrency int
	Timeout     time.Duration
}

type Result struct {
	StatusCode int
	Duration   time.Duration
	Err        error
}

type Runner struct {
	client *http.Client
}

func New(timeout time.Duration) *Runner {
	return &Runner{
		client: &http.Client{Timeout: timeout},
	}
}

func (r *Runner) Run(ctx context.Context, cfg Config) <-chan Result {
	results := make(chan Result)
	jobs := make(chan struct{})

	go func() {
		defer close(jobs)
		for i := 0; i < cfg.Requests; i++ {
			select {
			case <-ctx.Done():
				return
			case jobs <- struct{}{}:
			}
		}
	}()

	workerDone := make(chan struct{}, cfg.Concurrency)
	for i := 0; i < cfg.Concurrency; i++ {
		go func() {
			defer func() { workerDone <- struct{}{} }()
			for range jobs {
				res := r.doRequest(ctx, cfg)
				select {
				case results <- res:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		for i := 0; i < cfg.Concurrency; i++ {
			<-workerDone
		}
		close(results)
	}()

	return results
}

func (r *Runner) doRequest(ctx context.Context, cfg Config) Result {
	started := time.Now()

	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, nil)
	if err != nil {
		return Result{Duration: time.Since(started), Err: err}
	}

	resp, err := r.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Result{Duration: time.Since(started), Err: err}
		}
		return Result{Duration: time.Since(started), Err: err}
	}
	defer resp.Body.Close()

	_, copyErr := io.Copy(io.Discard, resp.Body)
	if copyErr != nil {
		return Result{
			StatusCode: resp.StatusCode,
			Duration:   time.Since(started),
			Err:        copyErr,
		}
	}

	return Result{
		StatusCode: resp.StatusCode,
		Duration:   time.Since(started),
	}
}
