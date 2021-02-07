package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/model"
)

// CoordsHandler godoc
func CoordsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, core.Coords())
}

// SunriseSunsetHandler sunrise/sunset godoc
func SunriseSunsetHandler(c *gin.Context) {
	sunrise, sunset, err := core.Coords().GetSunriseSunset()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.NewAPIError(err))
		return
	}

	c.JSON(http.StatusOK, struct {
		Sunrise time.Time
		Sunset  time.Time
	}{
		Sunrise: sunrise,
		Sunset:  sunset,
	})
}
