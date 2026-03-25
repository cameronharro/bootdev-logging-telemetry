package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"boot.dev/linko/internal/build"
	"boot.dev/linko/internal/linkoerr"
	pkgerr "github.com/pkg/errors"
)

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}
type multiError interface {
	error
	Unwrap() []error
}

func errorAttrs(err error) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("message", err.Error()),
	}
	attrs = append(attrs, linkoerr.Attrs(err)...)
	if stackErr, ok := errors.AsType[stackTracer](err); ok {
		attrs = append(
			attrs,
			slog.String("stack_trace", fmt.Sprintf("%+v", stackErr.StackTrace())),
		)
	}
	return attrs
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		if err, ok := a.Value.Any().(error); ok {
			if multiErr, ok := errors.AsType[multiError](err); ok {
				var errAttrs []slog.Attr
				for i, e := range multiErr.Unwrap() {
					errAttrs = append(errAttrs, slog.GroupAttrs(fmt.Sprintf("error_%d", i+1), errorAttrs(e)...))
				}
				return slog.GroupAttrs("errors", errAttrs...)
			}

			return slog.GroupAttrs("error", errorAttrs(err)...)
		}
	}
	return a
}

func newLogger(logFileName string) (logger *slog.Logger, closeFunc func() error, err error) {
	noop := func() error { return nil }
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
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
		ReplaceAttr: replaceAttr,
	})
	logger = slog.New(slog.NewMultiHandler(
		debugHandler,
		infoHandler,
	))
	logger = logger.With(
		slog.String("git_sha", build.GitSHA),
		slog.String("build_time", build.BuildTime),
	)
	return logger, func() error { return bufferedFile.Flush() }, nil
}
