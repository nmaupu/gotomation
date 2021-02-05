package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// GoogleConfig is the google configuration used to access the Google API
type GoogleConfig struct {
	Config        *oauth2.Config
	Token         *oauth2.Token
	TokenFilePath string
}

// LoadTokenFromCache loads Google token from cache file
func (g *GoogleConfig) LoadTokenFromCache() error {
	f, err := os.Open(g.TokenFilePath)
	if err != nil {
		return errors.Wrapf(err, "Unable to open Google's token cache file %s", g.TokenFilePath)
	}

	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		return errors.Wrapf(err, "Unable to decode Google token cache file %s", g.TokenFilePath)
	}

	g.Token = tok
	return nil
}

// GoogleWebTokenHandler is the handler to fill a token gotten after using the auth url
func (g GoogleConfig) GoogleWebTokenHandler(c *gin.Context) {
	l := logging.NewLogger("GoogleWebTokenHandler")
	if g.Config == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.NewAPIError(fmt.Errorf("Google config is not set")))
		return
	}

	// Getting GET parameters
	getParameters := struct {
		AuthCode string `form:"auth_code" json:"auth_code"`
	}{}
	_ = c.MustBindWith(&getParameters, binding.Form) // ignore errors

	if getParameters.AuthCode == "" {
		l.Error().Err(fmt.Errorf("auth_code parameter is not set")).Msg("Unable to valide Google auth code")
	}

	tok, err := g.Config.Exchange(context.TODO(), getParameters.AuthCode)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.NewAPIError(err))
	}

	g.Token = tok
	//l.Trace().Str("token", fmt.Sprintf("%+v", g.Token)).Msg("Personal token got from Google")

	// Saving token to a local file

	l.Debug().Msgf("Token got from Google, caching to local file %s", g.TokenFilePath)
	f, err := os.OpenFile(g.TokenFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		l.Error().Err(err).Msg("Unable to store token file")
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.NewAPIError(err))
		return
	}
	defer f.Close()

	json.NewEncoder(f).Encode(g.Token)

	c.Status(http.StatusNoContent)
}
