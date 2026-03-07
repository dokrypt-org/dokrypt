package common

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

func ParseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func SetupLogger(level string, jsonFormat bool, writer io.Writer) {
	if writer == nil {
		writer = os.Stderr
	}

	slogLevel := ParseLogLevel(level)
	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func IsCI() bool {
	return os.Getenv("DOKRYPT_CI") == "true" || os.Getenv("CI") == "true"
}

func NoColor() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("DOKRYPT_NO_COLOR") == "true" || IsCI()
}
