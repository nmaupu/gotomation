package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/app"
)

type health struct {
	Version   string `json:"version"`
	BuildDate string `json:"BuildDate"`
	Status    string `json:"status"`
}

// HealthHandler godoc
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, health{
		Version:   app.ApplicationVersion,
		BuildDate: app.BuildDate,
		Status:    http.StatusText(http.StatusOK),
	})
}
