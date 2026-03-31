package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boot.dev/linko/internal/store"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()

	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	logger, closeFunc, err := newLogger(os.Getenv("LINKO_LOG_FILE"))
	if err != nil {
		slog.Error(
			"failed to initialize logger: %v",
			slog.String("err", err.Error()),
		)
		return 1
	}
	defer func() {
		if err := closeFunc(); err != nil {
			slog.Error(err.Error())
		}
	}()

	tracerCloser, err := initTracing(context.Background())
	if err != nil {
		logger.Error(
			"failed to initialize tracer",
			slog.Any("error", err),
		)
		return 1
	}
	defer func() {
		if err := tracerCloser(context.Background()); err != nil {
			slog.Error(err.Error())
		}
	}()

	st, err := store.New(dataDir, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create store: %v\n", err))
		return 1
	}

	s := newServer(*st, httpPort, logger, cancel)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		logger.Error(fmt.Sprintf("failed to shutdown server: %v", err))
		return 1
	}
	if serverErr != nil {
		logger.Error(fmt.Sprintf("server error: %v\n", serverErr))
		return 1
	}
	return 0
}
