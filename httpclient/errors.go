package httpclient

import "fmt"

var _ error = (*ErrorStatusNotOK)(nil)

// ErrorStatusNotOK is used to notify caller that a returned status is not considered OK
type ErrorStatusNotOK struct {
	Message string
	Status  int
}

func NewErrorStatusNotOK(status int) *ErrorStatusNotOK {
	return &ErrorStatusNotOK{
		Message: fmt.Sprintf("HTTP response code is not ok, got=%d", status),
		Status:  status,
	}
}

func (e ErrorStatusNotOK) Error() string {
	return e.Message
}
