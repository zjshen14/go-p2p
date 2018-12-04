package p2p

import (
	"os"

	"github.com/rs/zerolog"
)

// Logger is the logger instance
var Logger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()
