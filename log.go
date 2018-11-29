package p2p

import (
	"os"

	"github.com/rs/zerolog"
)

var Logger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()
