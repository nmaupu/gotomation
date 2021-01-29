package app

import "sync"

var (
	// RoutinesWG registers all go routines of gotomation service
	RoutinesWG = sync.WaitGroup{}
)
