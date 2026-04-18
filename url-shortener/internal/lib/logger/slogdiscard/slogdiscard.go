package slogdiscard

import (
	"io"
	"log/slog"
)

func NewDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
