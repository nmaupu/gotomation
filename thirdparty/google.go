package thirdparty

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/mitchellh/go-homedir"
	"github.com/nmaupu/gotomation/logging"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	once                  sync.Once
	googleConfigSingleton googleConfig
	onceErr               error
)

// GoogleConfig is the interface to interact with the Google API
// Deprecated: not used for anything
type GoogleConfig interface {
	GetAuthURL() string
	GetTokenFromWeb(authCode string) error
	GetClient() (*http.Client, error)
}

// GoogleConfig is the configuration to access the Google API
type googleConfig struct {
	config         *oauth2.Config
	accessToken    *oauth2.Token
	tokenCachePath string
}

// GetGoogleConfig returns the configured GoogleConfig object
// Deprecated: not used for anything
func GetGoogleConfig() GoogleConfig {
	return &googleConfigSingleton
}

// InitGoogleConfig returns a new pointer to a GoogleConfig object
func InitGoogleConfig(credsFilePath string, scopes ...string) error {
	l := logging.NewLogger("InitGoogleConfig")

	if credsFilePath == "" {
		return fmt.Errorf("Google credentials file is not set, nothing to do")
	}

	once.Do(func() {
		l.Info().Msg("Initializing Google creds config")
		var hdir string
		hdir, onceErr = homedir.Dir()
		if onceErr != nil {
			return
		}
		expHomedir, onceErr := homedir.Expand(hdir)
		if onceErr != nil {
			return
		}
		cacheFile := fmt.Sprintf("%s/.gotomation-google-token.json", expHomedir)

		b, onceErr := ioutil.ReadFile(credsFilePath)
		if onceErr != nil {
			return
		}

		googleConfigSingleton = googleConfig{
			tokenCachePath: cacheFile,
		}
		googleConfigSingleton.config, onceErr = google.ConfigFromJSON(b, scopes...)
	})

	return onceErr
}

// LoadTokenFromCache loads token from cache file
func loadTokenFromCache(cacheFile string) (*oauth2.Token, error) {
	f, err := os.Open(cacheFile)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to open Google's token cache file %s", cacheFile)
	}
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to decode Google token cache file %s", cacheFile)
	}

	return tok, nil
}

// SaveTokenToCache saves current token to a cache file
func (g *googleConfig) saveTokenToCache() error {
	if g.accessToken == nil {
		return fmt.Errorf("Unable to save, token is not set")
	}

	f, err := os.OpenFile(g.tokenCachePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(g.accessToken); err != nil {
		return err
	}

	return nil
}

// GetTokenFromWeb gets a token from a consent screen
func (g *googleConfig) GetTokenFromWeb(authCode string) error {
	if g.config == nil {
		return fmt.Errorf("config is nil, cannot proceed")
	}

	tok, err := g.config.Exchange(context.Background(), authCode)
	if err != nil {
		return err
	}
	g.accessToken = tok
	if err := g.saveTokenToCache(); err != nil {
		return errors.Wrapf(err, "Token is set but unable to save to cache file %s", g.tokenCachePath)
	}

	return nil
}

// GetAuthURL returns an authentication URL used to get an auth code
func (g *googleConfig) GetAuthURL() string {
	if g.config == nil {
		return ""
	}

	return g.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func (g *googleConfig) GetClient() (*http.Client, error) {
	l := logging.NewLogger("GoogleConfig.GetClient")
	l.Debug().Msg("Getting google http client")

	tok, err := loadTokenFromCache(g.tokenCachePath)
	if err != nil || tok == nil {
		return nil, fmt.Errorf("Unable to load token from cache, use %s to create a token", g.GetAuthURL())
	}

	l.Debug().
		Str("token", fmt.Sprintf("***%s", tok.AccessToken[len(tok.AccessToken)-9:len(tok.AccessToken)-1])).
		Msg("Load token from cache ok")
	g.accessToken = tok
	return g.config.Client(context.Background(), g.accessToken), nil
}
