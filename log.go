package p2p

import (
	"os"

	"github.com/rs/zerolog"
)

// logger is the logger instance
var logger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()

// Logger returns the logger
func Logger() *zerolog.Logger { return &logger }

// SetLogger sets the logger
func SetLogger(l *zerolog.Logger) {
	logger = *l
}
