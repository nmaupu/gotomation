package model

// APIError is the representation of an error usable in json
type APIError struct {
	Error string `json:"error"`
}

// NewAPIError returns a new APIError object
func NewAPIError(err error) APIError {
	if err == nil {
		return APIError{}
	}

	return APIError{
		Error: err.Error(),
	}
}
