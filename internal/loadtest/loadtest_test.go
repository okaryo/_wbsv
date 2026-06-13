package loadtest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunExecutesRequestsConcurrently(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte("created"))
	}))
	defer server.Close()

	result, err := Run(context.Background(), Config{
		URL:         server.URL,
		Requests:    10,
		Concurrency: 3,
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Requests != 10 {
		t.Fatalf("Requests = %d, want 10", result.Requests)
	}
	if result.Completed != 10 {
		t.Fatalf("Completed = %d, want 10", result.Completed)
	}
	if result.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", result.Failed)
	}
	if result.StatusCounts[201] != 10 {
		t.Fatalf("201 count = %d, want 10", result.StatusCounts[201])
	}
	if result.BytesRead != int64(10*len("created")) {
		t.Fatalf("BytesRead = %d, want response bytes", result.BytesRead)
	}
	if result.MinLatency <= 0 || result.MaxLatency <= 0 || result.AvgLatency <= 0 {
		t.Fatalf("latencies = min %s max %s avg %s, want positive values", result.MinLatency, result.MaxLatency, result.AvgLatency)
	}
}

func TestRunRecordsRequestFailures(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), Config{
		URL:         "http://127.0.0.1:1",
		Requests:    2,
		Concurrency: 1,
		Timeout:     10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Completed != 0 {
		t.Fatalf("Completed = %d, want 0", result.Completed)
	}
	if result.Failed != 2 {
		t.Fatalf("Failed = %d, want 2", result.Failed)
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Config{})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Run() error = %v, want ErrInvalidConfig", err)
	}
}

func TestSortedStatusCodes(t *testing.T) {
	t.Parallel()

	got := SortedStatusCodes(map[int]int{
		500: 1,
		200: 2,
		404: 3,
	})
	want := []int{200, 404, 500}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("codes = %+v, want %+v", got, want)
		}
	}
}
