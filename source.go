package searchwire

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

const rrfK = 60

type source interface {
	name() string
	search(context.Context, *fetcher, string, int) ([]sourceResult, error)
}

type sourceResult struct {
	title   string
	url     string
	snippet string
	rank    int
}

type fetcher struct {
	client          HTTPClient
	userAgent       string
	maxResponseSize int64
}

func newFetcher(client HTTPClient, userAgent string, maxSize int64) *fetcher {
	return &fetcher{
		client:          client,
		userAgent:       userAgent,
		maxResponseSize: maxSize,
	}
}

func (f *fetcher) get(ctx context.Context, rawURL, accept string, allowedTypes []string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, f.maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if int64(len(body)) > f.maxResponseSize {
		return nil, ErrResponseTooLarge
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	if err := validateContentType(resp.Header.Get("Content-Type"), allowedTypes); err != nil {
		return nil, err
	}
	return body, nil
}

func validateContentType(header string, allowed []string) error {
	if len(allowed) == 0 || strings.TrimSpace(header) == "" {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil || mediaType == "" {
		return nil
	}
	for _, allowedType := range allowed {
		if mediaType == allowedType {
			return nil
		}
		if allowedType == "application/json" && (strings.HasSuffix(mediaType, "+json") || mediaType == "application/json") {
			return nil
		}
		if allowedType == "text/html" && strings.HasPrefix(mediaType, "text/html") {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrUnexpectedContentType, mediaType)
}

func isSafeResultURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(raw), "javascript:") {
		return false
	}
	if strings.HasPrefix(raw, "//") {
		return false
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return false
	}
	return true
}

func defaultSources() []source {
	return []source{
		braveSource{},
		startpageSource{},
		wikipediaSource{},
	}
}

func sourceNames(sources []source) []string {
	names := make([]string, len(sources))
	for i, src := range sources {
		names[i] = src.name()
	}
	return names
}

func orderSourceErrors(failures []SourceError, sources []source) []SourceError {
	if len(failures) <= 1 {
		return failures
	}
	order := make(map[string]int, len(sources))
	for i, src := range sources {
		order[src.name()] = i
	}
	ordered := append([]SourceError(nil), failures...)
	for i := range ordered {
		for j := i + 1; j < len(ordered); j++ {
			if order[ordered[j].Source] < order[ordered[i].Source] {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}
	return ordered
}

// newSearcherWithSources is used by package tests to inject fake sources.
func newSearcherWithSources(sources []source, options ...Option) *Searcher {
	s := New(options...)
	s.sources = sources
	return s
}
