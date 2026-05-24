package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/okaryo/_wbsv/internal/tcpserver"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "TCP listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "wbsv: ", log.LstdFlags|log.Lmicroseconds)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &tcpserver.Server{
		Addr:   *addr,
		Logger: logger,
	}

	if err := server.ListenAndServe(ctx); err != nil {
		logger.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
