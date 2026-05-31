package httpserver

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/okaryo/_wbsv/internal/http1"
)

const (
	defaultMaxLine    = 8192
	defaultMaxHeaders = 100
	defaultMaxBody    = 1 << 20
)

// Handler reads HTTP requests from a connection and writes responses.
type Handler struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	MaxLine      int
	MaxHeaders   int
	MaxBody      int64
	Logger       *log.Logger
}

// ServeConn handles HTTP/1.x requests until the connection should close.
func (h *Handler) ServeConn(conn net.Conn) {
	reader := http1.NewLineReader(conn, h.maxLine())

	for {
		if h.ReadTimeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(h.ReadTimeout)); err != nil {
				h.logf("set read deadline error for %s: %v", conn.RemoteAddr(), err)
				return
			}
		}

		h.logf("waiting for HTTP request from %s", conn.RemoteAddr())
		request, err := http1.ReadRequest(reader, http1.RequestLimits{
			MaxHeaders: h.maxHeaders(),
			MaxBody:    h.maxBody(),
		})

		response, closeAfterResponse := h.responseForRequest(request, err)
		if closeAfterResponse {
			response.Headers = append(response.Headers, http1.HeaderField{Name: "Connection", Value: "close"})
		}

		if h.WriteTimeout > 0 {
			if err := conn.SetWriteDeadline(time.Now().Add(h.WriteTimeout)); err != nil {
				h.logf("set write deadline error for %s: %v", conn.RemoteAddr(), err)
				return
			}
		}

		if err := http1.WriteResponse(conn, response); err != nil {
			h.logf("write response error for %s: %v", conn.RemoteAddr(), err)
			return
		}
		h.logf("handled HTTP request from %s", conn.RemoteAddr())

		if closeAfterResponse {
			return
		}
	}
}

func (h *Handler) responseForRequest(request http1.Request, err error) (http1.Response, bool) {
	if err != nil {
		h.logf("request parse error: %v", err)

		var netErr net.Error
		switch {
		case errors.As(err, &netErr) && netErr.Timeout():
			return http1.ErrorResponse(408, "request timeout"), true
		case errors.Is(err, http1.ErrBodyTooLarge):
			return http1.ErrorResponse(413, "content too large"), true
		case errors.Is(err, http1.ErrUnsupportedTransferEncoding):
			return http1.ErrorResponse(501, "transfer encoding is not implemented"), true
		default:
			return http1.ErrorResponse(400, "bad request"), true
		}
	}

	body := []byte("hello from _wbsv\n")
	response := http1.Response{
		StatusCode: 200,
		Headers: []http1.HeaderField{
			{Name: "X-WBSV-Method", Value: request.RequestLine.Method},
			{Name: "X-WBSV-Target", Value: request.RequestLine.RequestTarget},
		},
		Body: body,
	}
	closeAfterResponse := shouldCloseConnection(request)
	return http1.WithContentType(response, "text/plain; charset=utf-8"), closeAfterResponse
}

func shouldCloseConnection(request http1.Request) bool {
	if http1.HeaderHasToken(request.Headers, "Connection", "close") {
		return true
	}
	return request.RequestLine.Version != "HTTP/1.1"
}

func (h *Handler) maxLine() int {
	if h.MaxLine > 0 {
		return h.MaxLine
	}
	return defaultMaxLine
}

func (h *Handler) maxHeaders() int {
	if h.MaxHeaders > 0 {
		return h.MaxHeaders
	}
	return defaultMaxHeaders
}

func (h *Handler) maxBody() int64 {
	if h.MaxBody > 0 {
		return h.MaxBody
	}
	return defaultMaxBody
}

func (h *Handler) logf(format string, args ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, args...)
	}
}
