package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

func newLogger(logFileName string) (logger *slog.Logger, closeFunc func() error, err error) {
	noop := func() error { return nil }
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	if logFileName == "" {
		return slog.New(debugHandler), noop, nil
	}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, noop, fmt.Errorf("failed to open log file: %w", err)
	}

	bufferedFile := bufio.NewWriterSize(logFile, 8192)
	infoHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger = slog.New(slog.NewMultiHandler(
		debugHandler,
		infoHandler,
	))
	return logger, func() error { return bufferedFile.Flush() }, nil
}
