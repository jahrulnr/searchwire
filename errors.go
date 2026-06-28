package searchwire

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEmptyQuery            = errors.New("searchwire: query is required")
	ErrNilSearcher           = errors.New("searchwire: nil searcher")
	ErrResponseTooLarge      = errors.New("searchwire: response exceeds configured limit")
	ErrUnexpectedContentType = errors.New("searchwire: unexpected content type")
)

// SearchError is returned when every built-in source fails.
type SearchError struct {
	Failures []SourceError
}

func (e *SearchError) Error() string {
	if e == nil || len(e.Failures) == 0 {
		return "searchwire: all sources failed"
	}
	parts := make([]string, len(e.Failures))
	for i, f := range e.Failures {
		parts[i] = f.Source + ": " + f.Error
	}
	return "searchwire: all sources failed: " + strings.Join(parts, "; ")
}

// SourceError records one source failure without aborting the whole search.
type SourceError struct {
	Source string
	Error  string
}

// HTTPError reports a non-2xx HTTP response from a source.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("searchwire: source returned %s", e.Status)
	}
	return fmt.Sprintf("searchwire: source returned %s: %s", e.Status, e.Body)
}
