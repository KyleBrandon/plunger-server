package health

import (
	"log/slog"
	"sync"
)

type Handler struct {
	logger *slog.Logger
	level  slog.Level
	mu     sync.RWMutex
}
