package searchwire

import (
	"errors"
	"fmt"
)

var (
	ErrResponseTooLarge      = errors.New("searchwire: response exceeds configured limit")
	ErrUnexpectedContentType = errors.New("searchwire: expected an application/json response")
)

// HTTPError reports a non-2xx response from the search backend.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("searchwire: search backend returned %s", e.Status)
	}
	return fmt.Sprintf("searchwire: search backend returned %s: %s", e.Status, e.Body)
}
