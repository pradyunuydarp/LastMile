package logging

import (
	"log/slog"
	"os"
)

// New returns a slog logger with a JSON handler that includes the service name.
// Callers can safely pass nil as the service to omit the attribute.
func New(service string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	if service == "" {
		return slog.New(handler)
	}

	return slog.New(handler).With("service", service)
}
