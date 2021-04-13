package model

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HassConfig represents a Home Assistant configuration to connect to
type HassConfig struct {
	URL                 url.URL
	Token               string
	HealthCheckEntities []HassEntity
}

// NewHTTPRequest creates a new http request to query Home Assistant's API
func (c HassConfig) NewHTTPRequest(method string, endpoint string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s://%s/%s/%s",
		c.URL.Scheme,
		c.URL.Host,
		c.URL.Path,
		endpoint)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	return req, nil
}
