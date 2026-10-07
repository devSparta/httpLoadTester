package stats

import (
	"sort"
	"time"

	"gload/internal/runner"
)

type Summary struct {
	Requested   int
	Completed   int
	Successful  int
	Failed      int
	Duration    time.Duration
	RPS         float64
	Average     time.Duration
	P50         time.Duration
	P95         time.Duration
	P99         time.Duration
	StatusCodes map[int]int
}

func Build(requested int, elapsed time.Duration, results []runner.Result) Summary {
	s := Summary{
		Requested:   requested,
		Completed:   len(results),
		Duration:    elapsed,
		StatusCodes: make(map[int]int),
	}

	latencies := make([]time.Duration, 0, len(results))
	var totalLatency time.Duration

	for _, result := range results {
		latencies = append(latencies, result.Duration)
		totalLatency += result.Duration

		if result.StatusCode != 0 {
			s.StatusCodes[result.StatusCode]++
		}

		if result.Err == nil && result.StatusCode >= 200 && result.StatusCode < 400 {
			s.Successful++
		} else {
			s.Failed++
		}
	}

	if s.Completed == 0 {
		return s
	}

	if elapsed > 0 {
		s.RPS = float64(s.Completed) / elapsed.Seconds()
	}

	s.Average = totalLatency / time.Duration(s.Completed)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	s.P50 = percentile(latencies, 50)
	s.P95 = percentile(latencies, 95)
	s.P99 = percentile(latencies, 99)

	return s
}

func percentile(values []time.Duration, p int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	index := (p*len(values) + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(values) {
		index = len(values)
	}

	return values[index-1]
}
