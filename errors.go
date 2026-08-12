package tuitui

import (
	"encoding/json"
	"fmt"
)

type APIError struct {
	Message  string
	Endpoint string
	Status   int
	ErrCode  int
	Response any
	Cause    error
}

func (e *APIError) Error() string {
	message := e.Message
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	if e.Response != nil {
		message += "; api response: " + formatAPIResponse(e.Response)
	}
	return message
}

func (e *APIError) Unwrap() error { return e.Cause }

func newAPIError(endpoint, message string, cause error) *APIError {
	return &APIError{Endpoint: endpoint, Message: fmt.Sprintf("%s %s", endpoint, message), Cause: cause}
}

func formatAPIResponse(response any) string {
	if value, ok := response.(string); ok {
		return value
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Sprint(response)
	}
	return string(encoded)
}
