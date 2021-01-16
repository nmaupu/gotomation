package model

var (
	_ HassAPIObject = (*HassResult)(nil)
)

// HassResult represents a Home Assistant message
type HassResult struct {
	ID          uint64      `json:"id"`
	Type        string      `json:"type"`
	HassVersion string      `json:"ha_version,omitempty"`
	Message     string      `json:"message,omitempty"`
	Success     bool        `json:"success,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	Error       struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
}

// GetID godoc
func (r HassResult) GetID() uint64 {
	return r.ID
}

// GetType godoc
func (r HassResult) GetType() string {
	return r.Type
}

// Duplicate godoc
func (r HassResult) Duplicate(id uint64) HassAPIObject {
	dup := r
	dup.ID = id
	return dup
}
