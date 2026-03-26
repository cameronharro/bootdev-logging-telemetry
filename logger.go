package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"boot.dev/linko/internal/build"
	"boot.dev/linko/internal/linkoerr"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	pkgerr "github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
)

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}
type multiError interface {
	error
	Unwrap() []error
}

func newLogger(logFileName string) (logger *slog.Logger, closeFunc func() error, err error) {
	var (
		handlers []slog.Handler
		closers  []func() error
	)

	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "error" {
			if err, ok := a.Value.Any().(error); ok {
				errorAttrs := func(err error) []slog.Attr {
					attrs := []slog.Attr{slog.String("message", strings.ToLower(err.Error()))}
					attrs = append(
						attrs,
						linkoerr.Attrs(err)...,
					)
					if stackErr, ok := errors.AsType[stackTracer](err); ok {
						attrs = append(
							attrs,
							slog.String("stack_trace", fmt.Sprintf("%+v", stackErr.StackTrace())),
						)
					}
					return attrs
				}

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

	handlers = append(handlers, tint.NewHandler(os.Stderr, &tint.Options{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
		NoColor:     !(isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())),
	}))

	if logFileName != "" {
		fileLogger := &lumberjack.Logger{
			Filename:   logFileName,
			MaxSize:    1,
			MaxAge:     28,
			MaxBackups: 10,
			LocalTime:  false,
			Compress:   true,
		}
		handlers = append(handlers, slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		}))
		closers = append(closers, func() error { return fileLogger.Close() })
	}

	addEnvContext := func(logger *slog.Logger) *slog.Logger {
		env := os.Getenv("ENV")
		hostname, _ := os.Hostname()
		return logger.With(
			slog.String("git_sha", build.GitSHA),
			slog.String("build_time", build.BuildTime),
			slog.String("env", env),
			slog.String("hostname", hostname),
		)
	}

	logger = addEnvContext(slog.New(slog.NewMultiHandler(handlers...)))

	closeFunc = func() error {
		var errs []error
		for _, closer := range closers {
			errs = append(errs, closer())
		}
		return errors.Join(errs...)
	}

	return logger, closeFunc, nil
}
