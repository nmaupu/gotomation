package routines

import (
	"sync"

	"github.com/nmaupu/gotomation/logging"
)

var (
	// runnables stores all runnable objects
	mutex     sync.Mutex
	runnables []Runnable
)

// Runnable represents an object which can be started or stopped
type Runnable interface {
	Start() error
	Stop()
	GetName() string
}

// AddRunnable adds Runnable objects to the list
func AddRunnable(r ...Runnable) {
	mutex.Lock()
	defer mutex.Unlock()
	runnables = append(runnables, r...)
}

// ResetRunnablesList empties Runnable objects' list
func ResetRunnablesList() {
	mutex.Lock()
	defer mutex.Unlock()
	runnables = make([]Runnable, 0)
}

// StartAllRunnables starts all registered Runnable objects
func StartAllRunnables() {
	l := logging.NewLogger("StartAllRunnables")
	mutex.Lock()
	defer mutex.Unlock()
	for _, r := range runnables {
		l.Info().
			Str("runnable", r.GetName()).
			Msg("Starting runnable")
		r.Start()
	}
}

// StopAllRunnables stops all registered Runnable objects
func StopAllRunnables() {
	l := logging.NewLogger("StopAllRunnables")
	mutex.Lock()
	defer mutex.Unlock()
	for _, r := range runnables {
		l.Info().
			Str("runnable", r.GetName()).
			Msg("Stopping runnable")
		r.Stop()
	}
}
