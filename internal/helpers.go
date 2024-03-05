package internal

import (
	"log/slog"
	"os"
)

func GetLogger(name string) *slog.Logger {
	h := slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		},
	)

	l := slog.New(h).With("logger", name)

	return l
}
