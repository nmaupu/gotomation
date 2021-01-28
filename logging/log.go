package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// DefaultLogLevel is the default log level
const DefaultLogLevel = "warn"

// Log is a wrapper for zerolog.Logger
type Log struct {
	l *zerolog.Logger
}

var (
	logger = Log{}
	// LogLevels are all the available log levels
	LogLevels = []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
)

// InitLogger inits the main logger
func InitLogger(w io.Writer) {
	writer := w
	if w == nil {
		writer = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	}

	l := zerolog.New(writer).With().Timestamp().Logger()
	logger.l = &l
}

// NewLogger returns a logger for a given component
func NewLogger(component string) zerolog.Logger {
	if logger.l == nil {
		InitLogger(nil)
	}

	return logger.l.With().Str("component", component).Logger()
}

// GetLogLevelsAsString returns log levels as a string ready to be displayed
func GetLogLevelsAsString() string {
	return strings.Join(LogLevels, ", ")
}

// SetVerbosity sets the global verbosity for all logs
func SetVerbosity(verbosity string) error {
	level, err := zerolog.ParseLevel(verbosity)
	if err != nil {
		return fmt.Errorf("Wrong verbosity %s, allowed values: %s", verbosity, GetLogLevelsAsString())
	}

	zerolog.SetGlobalLevel(level)
	return nil
}
