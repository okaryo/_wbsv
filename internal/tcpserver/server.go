package tcpserver

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

const bufferSize = 4096

// Server accepts raw TCP connections and echoes received bytes back to clients.
type Server struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Logger       *log.Logger
}

// ListenAndServe starts listening on s.Addr and serves accepted connections.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listenerConfig := net.ListenConfig{}
	listener, err := listenerConfig.Listen(ctx, "tcp", s.Addr)
	if err != nil {
		return err
	}

	return s.Serve(ctx, listener)
}

// Serve accepts connections from listener until the context is canceled or an
// unrecoverable listener error occurs.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	defer listener.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-done:
		}
	}()

	s.logf("listening on %s", listener.Addr())

	for {
		s.logf("waiting for a connection")
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	s.logf("accepted connection from %s", conn.RemoteAddr())

	buf := make([]byte, bufferSize)
	for {
		if s.ReadTimeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(s.ReadTimeout)); err != nil {
				s.logf("set read deadline error for %s: %v", conn.RemoteAddr(), err)
				return
			}
		}

		s.logf("waiting for bytes from %s", conn.RemoteAddr())
		n, err := conn.Read(buf)
		if n > 0 {
			s.logf("read %d bytes from %s", n, conn.RemoteAddr())

			if s.WriteTimeout > 0 {
				if err := conn.SetWriteDeadline(time.Now().Add(s.WriteTimeout)); err != nil {
					s.logf("set write deadline error for %s: %v", conn.RemoteAddr(), err)
					return
				}
			}

			written, writeErr := conn.Write(buf[:n])
			if writeErr != nil {
				var netErr net.Error
				if errors.As(writeErr, &netErr) && netErr.Timeout() {
					s.logf("write timeout for %s", conn.RemoteAddr())
				} else {
					s.logf("write error for %s: %v", conn.RemoteAddr(), writeErr)
				}
				return
			}
			s.logf("wrote %d bytes to %s", written, conn.RemoteAddr())
		}

		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				s.logf("read timeout for %s", conn.RemoteAddr())
			} else if !errors.Is(err, io.EOF) {
				s.logf("read error for %s: %v", conn.RemoteAddr(), err)
			}
			s.logf("closed connection from %s", conn.RemoteAddr())
			return
		}
	}
}

func (s *Server) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}
