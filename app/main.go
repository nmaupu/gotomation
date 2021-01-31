package app

import (
	"sync"
)

var (
	// RoutinesWG registers all go routines of gotomation service
	RoutinesWG = sync.WaitGroup{}

	// ApplicationVersion is the version of the binary
	ApplicationVersion string
	// BuildDate is the date when the binary was built
	BuildDate string
)
