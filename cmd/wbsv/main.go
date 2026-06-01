package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/okaryo/_wbsv/internal/httpserver"
	"github.com/okaryo/_wbsv/internal/tcpserver"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "TCP listen address")
	readTimeout := flag.Duration("read-timeout", 30*time.Second, "maximum time to wait for bytes from a connected client")
	writeTimeout := flag.Duration("write-timeout", 30*time.Second, "maximum time to wait while writing bytes to a connected client")
	gracefulTimeout := flag.Duration("graceful-timeout", 5*time.Second, "maximum time to wait for active connections during shutdown before force closing them")
	maxLine := flag.Int("max-line", 8192, "maximum HTTP line length")
	maxHeaders := flag.Int("max-headers", 100, "maximum number of HTTP headers")
	maxBody := flag.Int64("max-body", 1<<20, "maximum HTTP request body size")
	flag.Parse()

	logger := log.New(os.Stdout, "wbsv: ", log.LstdFlags|log.Lmicroseconds)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpHandler := &httpserver.Handler{
		ReadTimeout:  *readTimeout,
		WriteTimeout: *writeTimeout,
		MaxLine:      *maxLine,
		MaxHeaders:   *maxHeaders,
		MaxBody:      *maxBody,
		Logger:       logger,
		Middleware: []httpserver.Middleware{
			httpserver.LoggingMiddleware(logger),
			httpserver.RecoveryMiddleware(logger),
		},
	}

	server := &tcpserver.Server{
		Addr:            *addr,
		ReadTimeout:     *readTimeout,
		WriteTimeout:    *writeTimeout,
		GracefulTimeout: *gracefulTimeout,
		Logger:          logger,
		ConnHandler:     httpHandler.ServeConn,
	}

	if err := server.ListenAndServe(ctx); err != nil {
		logger.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
