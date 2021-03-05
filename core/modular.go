package core

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Modular is an interface that will implement a check function
type Modular interface {
	Automate
	Check()
	GetInterval() time.Duration
	GinHandler(c *gin.Context)
}
