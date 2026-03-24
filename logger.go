package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"

	"boot.dev/linko/internal/linkoerr"
	pkgerr "github.com/pkg/errors"
)

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

func logErrorStack(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		if err, ok := a.Value.Any().(error); ok {
			log.Println("switching on error type")
			if stackError, ok := errors.AsType[stackTracer](err); ok {
				log.Println("stackerr")
				return slog.GroupAttrs(
					"error",
					slog.String("message", stackError.Error()),
					slog.String("stack_trace", fmt.Sprintf("%+v", stackError.StackTrace())),
				)
			}
			if attrErr, ok := errors.AsType[*linkoerr.ErrWithAttrs](err); ok {
				log.Println("attrErr")
				return slog.GroupAttrs(
					"error",
					attrErr.Attrs()...,
				)
			}
			log.Println("default")
			return slog.String("error", fmt.Sprintf("%+v", err))
		}
	}
	return a
}

func newLogger(logFileName string) (logger *slog.Logger, closeFunc func() error, err error) {
	noop := func() error { return nil }
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: logErrorStack,
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
		Level:       slog.LevelInfo,
		ReplaceAttr: logErrorStack,
	})
	logger = slog.New(slog.NewMultiHandler(
		debugHandler,
		infoHandler,
	))
	return logger, func() error { return bufferedFile.Flush() }, nil
}
