package httpspi

import "fmt"

// Error represents a general HTTP client error.
type Error struct {
	Message string
}

func (e Error) Error() string {
	return "http: " + e.Message
}

// StatusError represents a non-2XX HTTP response.
// Returned by Client.Request when the server responds with a non-success status code.
type StatusError struct {
	StatusCode int
	Status     string
}

func (e StatusError) Error() string {
	return fmt.Sprintf("http: server responded with status %d", e.StatusCode)
}
