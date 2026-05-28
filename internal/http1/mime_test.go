package http1

import "testing"

func TestContentTypeByPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{path: "index.html", want: "text/html; charset=utf-8"},
		{path: "/assets/app.CSS", want: "text/css; charset=utf-8"},
		{path: "data.json", want: "application/json"},
		{path: "image.png", want: "image/png"},
		{path: "unknown.bin", want: "application/octet-stream"},
		{path: "no-extension", want: "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got := ContentTypeByPath(tt.path)
			if got != tt.want {
				t.Fatalf("ContentTypeByPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestWithContentTypeAddsHeader(t *testing.T) {
	t.Parallel()

	response := WithContentType(Response{
		StatusCode: 200,
		Body:       []byte("hello"),
	}, "text/plain; charset=utf-8")

	if len(response.Headers) != 1 {
		t.Fatalf("headers len = %d, want 1", len(response.Headers))
	}
	if response.Headers[0] != (HeaderField{Name: "Content-Type", Value: "text/plain; charset=utf-8"}) {
		t.Fatalf("header = %#v, want Content-Type text/plain", response.Headers[0])
	}
}

func TestWithContentTypeReplacesExistingHeader(t *testing.T) {
	t.Parallel()

	response := WithContentType(Response{
		StatusCode: 200,
		Headers: []HeaderField{
			{Name: "Content-Type", Value: "text/plain"},
			{Name: "content-type", Value: "application/json"},
			{Name: "X-Test", Value: "ok"},
		},
	}, "text/html; charset=utf-8")

	want := []HeaderField{
		{Name: "Content-Type", Value: "text/html; charset=utf-8"},
		{Name: "X-Test", Value: "ok"},
	}
	if len(response.Headers) != len(want) {
		t.Fatalf("headers len = %d, want %d", len(response.Headers), len(want))
	}
	for i := range want {
		if response.Headers[i] != want[i] {
			t.Fatalf("headers[%d] = %#v, want %#v", i, response.Headers[i], want[i])
		}
	}
}
