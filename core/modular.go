package core

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Modular is an interface that will implement a check function
type Modular interface {
	Check()
	GetInterval() time.Duration
	IsEnabled() bool
	GetName() string
	GinHandler(c *gin.Context)
}
