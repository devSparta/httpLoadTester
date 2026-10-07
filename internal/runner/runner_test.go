package runner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCompletesRequestedNumberOfRequests(t *testing.T) {
	var hits atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := Config{
		URL:         server.URL,
		Method:      http.MethodGet,
		Requests:    25,
		Concurrency: 5,
		Timeout:     time.Second,
	}

	r := New(cfg.Timeout)
	results := make([]Result, 0, cfg.Requests)
	for result := range r.Run(context.Background(), cfg) {
		results = append(results, result)
	}

	if len(results) != cfg.Requests {
		t.Fatalf("results: got %d, want %d", len(results), cfg.Requests)
	}
	if hits.Load() != int64(cfg.Requests) {
		t.Fatalf("server hits: got %d, want %d", hits.Load(), cfg.Requests)
	}

	for i, result := range results {
		if result.Err != nil {
			t.Fatalf("result %d returned error: %v", i, result.Err)
		}
		if result.StatusCode != http.StatusNoContent {
			t.Fatalf("result %d status: got %d, want %d", i, result.StatusCode, http.StatusNoContent)
		}
	}
}

func TestRunRespectsConcurrencyLimit(t *testing.T) {
	var current atomic.Int64
	var maximum atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := current.Add(1)
		defer current.Add(-1)

		for {
			old := maximum.Load()
			if n <= old || maximum.CompareAndSwap(old, n) {
				break
			}
		}

		time.Sleep(25 * time.Millisecond)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	cfg := Config{
		URL:         server.URL,
		Method:      http.MethodGet,
		Requests:    12,
		Concurrency: 3,
		Timeout:     time.Second,
	}

	r := New(cfg.Timeout)
	for result := range r.Run(context.Background(), cfg) {
		if result.Err != nil {
			t.Fatalf("request returned error: %v", result.Err)
		}
	}

	if maximum.Load() > int64(cfg.Concurrency) {
		t.Fatalf("max concurrency: got %d, limit %d", maximum.Load(), cfg.Concurrency)
	}
}
