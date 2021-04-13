package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/pkg/errors"
)

// SimpleClient is a client to make standard HTTP requests
type SimpleClient interface {
	GetEntities(domain string, name string) ([]model.HassEntity, error)
	GetEntity(domain string, name string) (model.HassEntity, error)
	CheckServerAPIHealth() bool
	CallService(entity model.HassEntity, service string, extraParams map[string]interface{}) error
}

type simpleClient struct {
	model.HassConfig
}

// NewSimpleClient returns a new SimpleClient object
func NewSimpleClient(hassConfig model.HassConfig) SimpleClient {
	return &simpleClient{
		HassConfig: hassConfig,
	}
}

// GetEntities returns entities matching criteria
// Regexp patterns can be used
func (c *simpleClient) GetEntities(domain string, name string) ([]model.HassEntity, error) {
	l := logging.NewLogger("SimpleClient.GetEntities")

	req, err := c.HassConfig.NewHTTPRequest(http.MethodGet, "states", nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if 200 != resp.StatusCode {
		return nil, fmt.Errorf("HTTP response code is not 200, got=%d", resp.StatusCode)
	}

	// Getting all states
	entities := make([]model.HassEntity, 0)
	states := make([]model.HassState, 0)
	err = json.Unmarshal(body, &states)
	if err != nil {
		return nil, err
	}

	// Filter using pattern
	patternDomain := domain
	patternName := name
	if domain == "" {
		patternDomain = `.*`
	}
	if name == "" {
		patternName = `.*`
	}
	pattern := fmt.Sprintf(`^%s\.%s$`, patternDomain, patternName)
	l.Trace().Str("pattern", pattern).Msg("Checking entities with pattern")

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Cannot compile regexp pattern %s", pattern))
	}
	for _, state := range states {
		if re.Match([]byte(state.EntityID)) {
			entity := model.NewHassEntity(state.EntityID)
			entity.State = state
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// GetEntity retrieves one entity given its domain and its name
// Regexp patterns can be used
func (c *simpleClient) GetEntity(domain string, name string) (model.HassEntity, error) {
	entities, err := c.GetEntities(domain, name)
	if err != nil {
		return model.HassEntity{}, err
	}

	if len(entities) == 0 {
		return model.HassEntity{}, fmt.Errorf("entity %s.%s not found", domain, name)
	}

	if len(entities) > 1 {
		return model.HassEntity{}, fmt.Errorf("too many entities (%d) found for %s.%s", len(entities), domain, name)
	}

	return entities[0], nil
}

// CheckServerAPIHealth verifies that the server is started and ready to serve requests (and that database is loaded)
func (c *simpleClient) CheckServerAPIHealth() bool {
	// Checking each provided entities
	var err error
	for _, e := range c.HassConfig.HealthCheckEntities {
		_, err = c.GetEntity(e.Domain, e.EntityID)
		if err != nil {
			return false
		}
	}
	return true
}

// CallService calls a service
func (c *simpleClient) CallService(entity model.HassEntity, service string, extraParams map[string]interface{}) error {
	l := logging.NewLogger("SimpleClient.CallService")
	l.Debug().
		Object("entity", entity).
		Str("service", service).
		Str("extra_params", fmt.Sprintf("%+v", extraParams)).
		Msg("Calling service")

	req, err := c.HassConfig.NewHTTPRequest(http.MethodPost, fmt.Sprintf("services/%s/%s", entity.Domain, service), nil)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"entity_id": entity.GetEntityIDFullName(),
	}
	for k, v := range extraParams {
		params[k] = v
	}

	reqBody, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP response code is not 200, got=%s", resp.Status)
	}

	return nil
}
