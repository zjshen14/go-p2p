package p2p

import (
	"os"
	"sync"

	"github.com/rs/zerolog"
)

// logger is the logger instance
var (
	_loggerMu sync.RWMutex
	_logger   = zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()
)

// Logger returns the logger
func Logger() *zerolog.Logger {
	_loggerMu.RLock()
	l := _logger
	_loggerMu.RUnlock()
	return &l
}

// SetLogger sets the logger
func SetLogger(l *zerolog.Logger) {
	_loggerMu.Lock()
	_logger = *l
	_loggerMu.Unlock()
}
