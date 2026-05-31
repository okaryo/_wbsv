package httpserver

import "testing"

func TestBufferedResponseWriterDefaultsStatusOnWrite(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	n, err := writer.Write([]byte("ok"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != 2 {
		t.Fatalf("written bytes = %d, want 2", n)
	}

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if string(response.Body) != "ok" {
		t.Fatalf("body = %q, want %q", string(response.Body), "ok")
	}
}

func TestBufferedResponseWriterKeepsFirstStatusCode(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	writer.WriteHeader(201)
	writer.WriteHeader(500)

	response := writer.Response()
	if response.StatusCode != 201 {
		t.Fatalf("status code = %d, want 201", response.StatusCode)
	}
}

func TestBufferedResponseWriterSetsAndAddsHeaders(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	writer.AddHeader("Set-Cookie", "a=1")
	writer.AddHeader("Set-Cookie", "b=2")
	writer.SetHeader("Content-Type", "text/plain")
	writer.SetHeader("content-type", "application/json")

	response := writer.Response()
	if len(response.Headers) != 3 {
		t.Fatalf("headers count = %d, want 3", len(response.Headers))
	}
	if response.Headers[0].Name != "Set-Cookie" || response.Headers[0].Value != "a=1" {
		t.Fatalf("first header = %+v, want Set-Cookie a=1", response.Headers[0])
	}
	if response.Headers[1].Name != "Set-Cookie" || response.Headers[1].Value != "b=2" {
		t.Fatalf("second header = %+v, want Set-Cookie b=2", response.Headers[1])
	}
	if response.Headers[2].Name != "Content-Type" || response.Headers[2].Value != "application/json" {
		t.Fatalf("third header = %+v, want Content-Type application/json", response.Headers[2])
	}
}
