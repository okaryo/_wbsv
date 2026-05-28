package http1

import (
	"path/filepath"
	"strings"
)

const defaultContentType = "application/octet-stream"

var contentTypesByExtension = map[string]string{
	".css":  "text/css; charset=utf-8",
	".gif":  "image/gif",
	".html": "text/html; charset=utf-8",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".js":   "text/javascript; charset=utf-8",
	".json": "application/json",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".txt":  "text/plain; charset=utf-8",
	".wasm": "application/wasm",
	".webp": "image/webp",
	".xml":  "application/xml",
}

// ContentTypeByPath returns a common MIME type for path based on its extension.
func ContentTypeByPath(path string) string {
	extension := strings.ToLower(filepath.Ext(path))
	if contentType, ok := contentTypesByExtension[extension]; ok {
		return contentType
	}
	return defaultContentType
}

// WithContentType returns a response with a Content-Type header.
func WithContentType(response Response, contentType string) Response {
	if contentType == "" {
		contentType = defaultContentType
	}

	headers := make([]HeaderField, 0, len(response.Headers)+1)
	replaced := false
	for _, header := range response.Headers {
		if strings.EqualFold(header.Name, "Content-Type") {
			if !replaced {
				headers = append(headers, HeaderField{Name: "Content-Type", Value: contentType})
				replaced = true
			}
			continue
		}
		headers = append(headers, header)
	}
	if !replaced {
		headers = append(headers, HeaderField{Name: "Content-Type", Value: contentType})
	}

	response.Headers = headers
	return response
}

// WithContentTypeForPath returns a response with Content-Type selected from path.
func WithContentTypeForPath(response Response, path string) Response {
	return WithContentType(response, ContentTypeByPath(path))
}
