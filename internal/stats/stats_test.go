package stats

import (
	"errors"
	"testing"
	"time"

	"gload/internal/runner"
)

func TestBuild(t *testing.T) {
	results := []runner.Result{
		{StatusCode: 200, Duration: 10 * time.Millisecond},
		{StatusCode: 204, Duration: 20 * time.Millisecond},
		{StatusCode: 500, Duration: 30 * time.Millisecond},
		{Duration: 40 * time.Millisecond, Err: errors.New("network error")},
	}

	s := Build(4, 2*time.Second, results)

	if s.Completed != 4 {
		t.Fatalf("completed: got %d, want 4", s.Completed)
	}
	if s.Successful != 2 {
		t.Fatalf("successful: got %d, want 2", s.Successful)
	}
	if s.Failed != 2 {
		t.Fatalf("failed: got %d, want 2", s.Failed)
	}
	if s.RPS != 2 {
		t.Fatalf("rps: got %.2f, want 2.00", s.RPS)
	}
	if s.Average != 25*time.Millisecond {
		t.Fatalf("average: got %s, want 25ms", s.Average)
	}
	if s.P50 != 20*time.Millisecond {
		t.Fatalf("p50: got %s, want 20ms", s.P50)
	}
	if s.P95 != 40*time.Millisecond {
		t.Fatalf("p95: got %s, want 40ms", s.P95)
	}
	if s.P99 != 40*time.Millisecond {
		t.Fatalf("p99: got %s, want 40ms", s.P99)
	}
	if s.StatusCodes[200] != 1 || s.StatusCodes[204] != 1 || s.StatusCodes[500] != 1 {
		t.Fatalf("unexpected status code counts: %#v", s.StatusCodes)
	}
}
