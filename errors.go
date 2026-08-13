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
	Response interface{}
	Cause    error
}

type wrappedError struct {
	message string
	cause   error
}

func (e *wrappedError) Error() string { return e.message + ": " + e.cause.Error() }
func (e *wrappedError) Unwrap() error { return e.cause }

func wrapError(message string, cause error) error {
	return &wrappedError{message: message, cause: cause}
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

func formatAPIResponse(response interface{}) string {
	if value, ok := response.(string); ok {
		return value
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Sprint(response)
	}
	return string(encoded)
}
