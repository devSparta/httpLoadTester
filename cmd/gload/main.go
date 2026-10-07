package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"gload/internal/runner"
	"gload/internal/stats"
)

type config struct {
	url         string
	method      string
	requests    int
	concurrency int
	timeout     time.Duration
}

func main() {
	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := runner.New(cfg.timeout)
	runCfg := runner.Config{
		URL:         cfg.url,
		Method:      cfg.method,
		Requests:    cfg.requests,
		Concurrency: cfg.concurrency,
		Timeout:     cfg.timeout,
	}

	fmt.Printf("Target:      %s\n", cfg.url)
	fmt.Printf("Method:      %s\n", cfg.method)
	fmt.Printf("Requests:    %d\n", cfg.requests)
	fmt.Printf("Concurrency: %d\n", cfg.concurrency)
	fmt.Printf("Timeout:     %s\n\n", cfg.timeout)

	started := time.Now()
	results := make([]runner.Result, 0, cfg.requests)
	for result := range r.Run(ctx, runCfg) {
		results = append(results, result)
	}
	elapsed := time.Since(started)

	summary := stats.Build(cfg.requests, elapsed, results)
	printSummary(summary)

	if errors.Is(ctx.Err(), context.Canceled) && summary.Completed < summary.Requested {
		fmt.Fprintln(os.Stderr, "\nRun interrupted.")
	}
}

func parseConfig() (config, error) {
	var cfg config

	flag.StringVar(&cfg.url, "url", "", "target URL")
	flag.StringVar(&cfg.method, "method", http.MethodGet, "HTTP method")
	flag.IntVar(&cfg.requests, "requests", 100, "total number of requests")
	flag.IntVar(&cfg.concurrency, "concurrency", 10, "number of concurrent workers")
	flag.DurationVar(&cfg.timeout, "timeout", 10*time.Second, "timeout per request")
	flag.Parse()

	cfg.method = strings.ToUpper(strings.TrimSpace(cfg.method))
	cfg.url = strings.TrimSpace(cfg.url)

	if cfg.url == "" {
		return config{}, errors.New("-url is required")
	}
	if cfg.requests <= 0 {
		return config{}, errors.New("-requests must be greater than zero")
	}
	if cfg.concurrency <= 0 {
		return config{}, errors.New("-concurrency must be greater than zero")
	}
	if cfg.concurrency > cfg.requests {
		cfg.concurrency = cfg.requests
	}
	if cfg.timeout <= 0 {
		return config{}, errors.New("-timeout must be greater than zero")
	}

	parsed, err := url.ParseRequestURI(cfg.url)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return config{}, errors.New("-url must be a valid absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return config{}, errors.New("-url must use http or https")
	}

	switch cfg.method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
	default:
		return config{}, fmt.Errorf("unsupported HTTP method %q", cfg.method)
	}

	return cfg, nil
}

func printSummary(s stats.Summary) {
	fmt.Println("Results")
	fmt.Println("-------")
	fmt.Printf("Requested:  %d\n", s.Requested)
	fmt.Printf("Completed:  %d\n", s.Completed)
	fmt.Printf("Successful: %d\n", s.Successful)
	fmt.Printf("Failed:     %d\n", s.Failed)
	fmt.Printf("Duration:   %s\n", formatDuration(s.Duration))
	fmt.Printf("RPS:        %.2f\n", s.RPS)
	fmt.Println()
	fmt.Println("Latency")
	fmt.Println("-------")
	fmt.Printf("avg: %s\n", formatDuration(s.Average))
	fmt.Printf("p50: %s\n", formatDuration(s.P50))
	fmt.Printf("p95: %s\n", formatDuration(s.P95))
	fmt.Printf("p99: %s\n", formatDuration(s.P99))

	if len(s.StatusCodes) == 0 {
		return
	}

	codes := make([]int, 0, len(s.StatusCodes))
	for code := range s.StatusCodes {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	fmt.Println()
	fmt.Println("Status codes")
	fmt.Println("------------")
	for _, code := range codes {
		fmt.Printf("%d: %d\n", code, s.StatusCodes[code])
	}
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d < time.Millisecond {
		return d.Round(time.Microsecond).String()
	}
	return d.Round(time.Millisecond).String()
}
