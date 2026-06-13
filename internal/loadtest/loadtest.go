package loadtest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

var ErrInvalidConfig = errors.New("invalid load test config")

// Config describes a small HTTP load test run.
type Config struct {
	URL               string
	Method            string
	Headers           http.Header
	Body              []byte
	Requests          int
	Concurrency       int
	Timeout           time.Duration
	DisableKeepAlives bool
}

// Result summarizes one load test run.
type Result struct {
	Requests     int
	Completed    int
	Failed       int
	BytesRead    int64
	StatusCounts map[int]int
	Duration     time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
	AvgLatency   time.Duration
}

type requestResult struct {
	statusCode int
	bytesRead  int64
	latency    time.Duration
	err        error
}

// Run executes a simple concurrent HTTP load test.
func Run(ctx context.Context, config Config) (Result, error) {
	if err := validateConfig(config); err != nil {
		return Result{}, err
	}
	if config.Method == "" {
		config.Method = http.MethodGet
	}

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			DisableKeepAlives: config.DisableKeepAlives,
		},
	}

	jobs := make(chan int)
	results := make(chan requestResult)

	var wg sync.WaitGroup
	for range config.Concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				results <- executeRequest(ctx, client, config)
			}
		}()
	}

	startedAt := time.Now()
	go func() {
		defer close(jobs)
		for i := range config.Requests {
			select {
			case <-ctx.Done():
				return
			case jobs <- i:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	result := Result{
		Requests:     config.Requests,
		StatusCounts: make(map[int]int),
	}
	var totalLatency time.Duration
	for requestResult := range results {
		if requestResult.err != nil {
			result.Failed++
		} else {
			result.Completed++
			result.BytesRead += requestResult.bytesRead
			result.StatusCounts[requestResult.statusCode]++
		}

		if result.MinLatency == 0 || requestResult.latency < result.MinLatency {
			result.MinLatency = requestResult.latency
		}
		if requestResult.latency > result.MaxLatency {
			result.MaxLatency = requestResult.latency
		}
		totalLatency += requestResult.latency
	}

	result.Duration = time.Since(startedAt)
	attempts := result.Completed + result.Failed
	if attempts > 0 {
		result.AvgLatency = totalLatency / time.Duration(attempts)
	}

	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, nil
}

func validateConfig(config Config) error {
	if config.URL == "" || config.Requests <= 0 || config.Concurrency <= 0 {
		return ErrInvalidConfig
	}
	return nil
}

func executeRequest(ctx context.Context, client *http.Client, config Config) requestResult {
	startedAt := time.Now()

	request, err := http.NewRequestWithContext(ctx, config.Method, config.URL, bytes.NewReader(config.Body))
	if err != nil {
		return requestResult{latency: time.Since(startedAt), err: err}
	}
	for name, values := range config.Headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}

	response, err := client.Do(request)
	if err != nil {
		return requestResult{latency: time.Since(startedAt), err: err}
	}
	defer response.Body.Close()

	bytesRead, err := io.Copy(io.Discard, response.Body)
	return requestResult{
		statusCode: response.StatusCode,
		bytesRead:  bytesRead,
		latency:    time.Since(startedAt),
		err:        err,
	}
}

// SortedStatusCodes returns response status codes in ascending order.
func SortedStatusCodes(counts map[int]int) []int {
	codes := make([]int, 0, len(counts))
	for code := range counts {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	return codes
}
