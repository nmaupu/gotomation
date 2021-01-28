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

var (
	logger *zerolog.Logger
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
	logger = &l
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

// Panic godoc
func Panic(f string) *zerolog.Event {
	if logger == nil {
		InitLogger(nil)
	}
	return logger.Panic().Str("func", f)
}

// Fatal godoc
func Fatal(f string) *zerolog.Event {
	if logger == nil {
		InitLogger(nil)
	}
	return logger.Fatal().Str("func", f)
}

// Warn godoc
func Warn(f string) *zerolog.Event {
	if logger == nil {
		InitLogger(nil)
	}
	return logger.Warn().Str("func", f)
}

// Error godoc
func Error(f string) *zerolog.Event {
	if logger == nil {
		InitLogger(nil)
	}
	return logger.Error().Str("func", f)
}

// Info godoc
func Info(f string) *zerolog.Event {
	if logger == nil {
		InitLogger(nil)
	}
	return logger.Info().Str("func", f)
}

// Debug godoc
func Debug(f string) *zerolog.Event {
	if logger == nil {
		InitLogger(nil)
	}
	return logger.Debug().Str("func", f)
}

// Trace godoc
func Trace(f string) *zerolog.Event {
	if logger == nil {
		InitLogger(nil)
	}
	return logger.Trace().Str("func", f)
}

// GetLogLevelsAsString returns log levels as a string ready to be displayed
func GetLogLevelsAsString() string {
	return strings.Join(LogLevels, ", ")
}
