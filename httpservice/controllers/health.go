package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/httpclient"
)

type health struct {
	Version   string `json:"version"`
	BuildDate string `json:"BuildDate"`
	Status    string `json:"status"`
}

func newHealth(status int) health {
	return health{
		Version:   app.ApplicationVersion,
		BuildDate: app.BuildDate,
		Status:    http.StatusText(status),
	}
}

// HealthHandler godoc
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, newHealth(http.StatusOK))
}

// HealthExHandler godoc
func HealthExHandler(c *gin.Context) {
	status := http.StatusOK
	if !httpclient.IsConnectedAndAuthenticated() {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, newHealth(status))
}
