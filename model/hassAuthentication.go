package model

var (
	_ HassAPIObject = (*HassAuthentication)(nil)
)

// HassAuthentication is the object to authenticate to the WebSocket API
type HassAuthentication struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token"`
}

// NewHassAuthentication returns a new HassAuthentication object
func NewHassAuthentication(token string) HassAuthentication {
	return HassAuthentication{
		Type:        "auth",
		AccessToken: token,
	}
}

// GetID godoc
func (e HassAuthentication) GetID() uint64 {
	return 0 // id is not used for authentication
}

// GetType godoc
func (e HassAuthentication) GetType() string {
	return e.Type
}

// Duplicate godoc
func (e HassAuthentication) Duplicate(id uint64) HassAPIObject {
	return NewHassAuthentication(e.AccessToken)
}
