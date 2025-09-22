package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Level        string
	Format       string
	SourceFolder string
}

type Logger struct {
	*slog.Logger
}

func New(cfg Config) *Logger {
	var lvl slog.Level
	switch cfg.Level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// Set more readable time format
				t := a.Value.Time().Format(time.TimeOnly)
				a.Value = slog.StringValue(t)
			}
			if a.Key == slog.SourceKey {
				src := a.Value.String()

				// Find the last occurrence of root path
				if startIndex := strings.LastIndex(src, cfg.SourceFolder); startIndex >= 0 {
					trimmed := src[startIndex+len(cfg.SourceFolder)+1:]

					// Fix " 74}" → ":74"
					trimmed = strings.Replace(trimmed, " ", ":", 1)
					trimmed = strings.TrimRight(trimmed, " }")

					a.Value = slog.StringValue(trimmed)
				}
			}

			return a
		},
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// set global default logger
	root := slog.New(handler)

	return &Logger{root}
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}
