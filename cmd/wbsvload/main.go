package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/okaryo/_wbsv/internal/loadtest"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:8080/hello", "target URL")
	method := flag.String("method", http.MethodGet, "HTTP method")
	requests := flag.Int("requests", 100, "total number of requests")
	concurrency := flag.Int("concurrency", 10, "number of concurrent workers")
	timeout := flag.Duration("timeout", 10*time.Second, "per-request timeout")
	body := flag.String("body", "", "request body")
	disableKeepAlives := flag.Bool("disable-keep-alives", false, "disable HTTP client keep-alive connection reuse")
	headers := headerFlags{}
	flag.Var(&headers, "header", "request header in Name: value form; may be repeated")
	flag.Parse()

	result, err := loadtest.Run(context.Background(), loadtest.Config{
		URL:               *url,
		Method:            *method,
		Headers:           headers.Header(),
		Body:              []byte(*body),
		Requests:          *requests,
		Concurrency:       *concurrency,
		Timeout:           *timeout,
		DisableKeepAlives: *disableKeepAlives,
	})

	printResult(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load test error: %v\n", err)
		os.Exit(1)
	}
}

type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlags) Set(value string) error {
	if _, _, ok := strings.Cut(value, ":"); !ok {
		return fmt.Errorf("header must be in Name: value form")
	}
	*h = append(*h, value)
	return nil
}

func (h *headerFlags) Header() http.Header {
	headers := make(http.Header)
	for _, raw := range *h {
		name, value, _ := strings.Cut(raw, ":")
		headers.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}
	return headers
}

func printResult(result loadtest.Result) {
	fmt.Printf("requests:    %d\n", result.Requests)
	fmt.Printf("completed:   %d\n", result.Completed)
	fmt.Printf("failed:      %d\n", result.Failed)
	fmt.Printf("bytes read:  %d\n", result.BytesRead)
	fmt.Printf("duration:    %s\n", result.Duration)
	fmt.Printf("latency min: %s\n", result.MinLatency)
	fmt.Printf("latency avg: %s\n", result.AvgLatency)
	fmt.Printf("latency max: %s\n", result.MaxLatency)
	fmt.Println("status:")
	for _, code := range loadtest.SortedStatusCodes(result.StatusCounts) {
		fmt.Printf("  %d: %d\n", code, result.StatusCounts[code])
	}
}
