package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

// SimpleClient is a client to make standard HTTP requests
type SimpleClient struct {
	model.HassConfig
}

// NewSimpleClient returns a new SimpleClient object
func NewSimpleClient(hassConfig model.HassConfig) *SimpleClient {
	return &SimpleClient{
		HassConfig: hassConfig,
	}
}

// GetEntities returns entities matching criteria
// Regexp patterns can be used
func (c *SimpleClient) GetEntities(domain string, name string) ([]model.HassEntity, error) {
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
	pattern := fmt.Sprintf("^%s\\.%s$", patternDomain, patternName)
	logging.Trace("SimpleClient.GetEntities").Str("pattern", pattern).Msg("Checking entities with pattern")

	re := regexp.MustCompile(pattern)
	for _, state := range states {
		if re.Match([]byte(state.EntityID)) {
			entity := model.HassEntity{}
			entity.EntityID = state.EntityID
			entity.State = state
			entity.Domain = strings.Split(state.EntityID, ".")[0]
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// GetEntity retrieves one entity given its domain and its name
// Regexp patterns can be used
func (c *SimpleClient) GetEntity(domain string, name string) (*model.HassEntity, error) {
	entities, err := c.GetEntities(domain, name)
	if err != nil {
		return nil, err
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("Entity %s.%s not found", domain, name)
	}

	if len(entities) > 1 {
		return nil, fmt.Errorf("Too many entities (%d) found for %s.%s", len(entities), domain, name)
	}

	return &entities[0], nil
}

// CheckServerAPIHealth verifies that the server is started and ready to serve requests (and that database is loaded)
func (c *SimpleClient) CheckServerAPIHealth() bool {
	// We suppose that if on of those entities are found, server is ready 🤷‍♂️
	_, err1 := c.GetEntity("light", "escalier_switch")
	_, err2 := c.GetEntity("light", "poutre_dimmer")
	return err1 == nil || err2 == nil
}

// CallService calls a service
func (c *SimpleClient) CallService(entity model.HassEntity, service string) error {
	req, err := c.HassConfig.NewHTTPRequest(http.MethodPost, fmt.Sprintf("services/%s/%s", entity.Domain, service), nil)
	if err != nil {
		return err
	}

	reqBody, err := json.Marshal(map[string]string{
		"entity_id": entity.GetEntityIDFullName(),
	})
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

	if 200 != resp.StatusCode {
		return fmt.Errorf("HTTP response code is not 200, got=%d", resp.StatusCode)
	}

	return nil
}
