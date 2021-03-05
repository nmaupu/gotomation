package core

import "sync"

var (
	_ Automate = (*automate)(nil)
)

// Automate is the base struct for all kind of automation stuff
type Automate interface {
	IsEnabled() bool
	Enable()
	Disable()
	GetName() string
}

type automate struct {
	Name          string `mapstructure:"name"`
	Disabled      bool   `mapstructure:"disabled"`
	mutexDisabled sync.Mutex
}

// IsDisabled godoc
func (a *automate) IsDisabled() bool {
	a.mutexDisabled.Lock()
	defer a.mutexDisabled.Unlock()
	return a.Disabled
}

func (a *automate) IsEnabled() bool {
	a.mutexDisabled.Lock()
	defer a.mutexDisabled.Unlock()
	return !a.Disabled
}

func (a *automate) Disable() {
	a.mutexDisabled.Lock()
	defer a.mutexDisabled.Unlock()
	a.Disabled = true
}

func (a *automate) Enable() {
	a.mutexDisabled.Lock()
	defer a.mutexDisabled.Unlock()
	a.Disabled = false
}

func (a *automate) GetName() string {
	return a.Name
}
