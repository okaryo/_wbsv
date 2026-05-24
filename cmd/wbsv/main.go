package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/okaryo/_wbsv/internal/tcpserver"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "TCP listen address")
	readTimeout := flag.Duration("read-timeout", 30*time.Second, "maximum time to wait for bytes from a connected client")
	flag.Parse()

	logger := log.New(os.Stdout, "wbsv: ", log.LstdFlags|log.Lmicroseconds)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &tcpserver.Server{
		Addr:        *addr,
		ReadTimeout: *readTimeout,
		Logger:      logger,
	}

	if err := server.ListenAndServe(ctx); err != nil {
		logger.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
