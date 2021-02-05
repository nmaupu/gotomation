package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/thirdparty"
)

// GoogleWebTokenHandler is the handler to fill a token gotten after using the auth url
func GoogleWebTokenHandler(c *gin.Context) {
	//l := logging.NewLogger("GoogleWebTokenHandler")

	// Getting GET parameters
	getParameters := struct {
		AuthCode string `form:"auth_code" json:"auth_code"`
	}{}
	_ = c.MustBindWith(&getParameters, binding.Form) // ignore errors

	if getParameters.AuthCode == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.NewAPIError(fmt.Errorf("auth_code parameter is not set")))
		return
	}

	err := thirdparty.GetGoogleConfig().GetTokenFromWeb(getParameters.AuthCode)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.NewAPIError(err))
		return
	}

	c.Status(http.StatusNoContent)
}
