package fritzboxmetrics

import (
	"errors"
	"fmt"
)

// ErrInvalidSOAPResponse will be thrown if we've got an invalid SOAP response
var ErrInvalidSOAPResponse = errors.New("invalid SOAP response")

// ResponseError contains the requested URI and the response status code
type ResponseError struct {
	URI        string
	StatusCode int
}

// Error returns a string representation
func (err *ResponseError) Error() string {
	return fmt.Sprintf("Unexpected status code %d for %s", err.StatusCode, err.URI)
}
