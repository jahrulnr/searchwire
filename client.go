package searchwire

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout          = 20 * time.Second
	defaultMaxResponseBytes = int64(1 << 20)
	defaultUserAgent        = "searchwire/0"
)

// HTTPClient is satisfied by *http.Client and lightweight test doubles.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientOption configures one SearXNG client.
type ClientOption struct {
	// URL is the root URL of a SearXNG instance or its /search endpoint.
	URL string
	// HTTPClient is optional. The default client has a 20-second timeout.
	HTTPClient HTTPClient
	// UserAgent defaults to searchwire/0.
	UserAgent string
	// MaxResponseBytes defaults to 1 MiB.
	MaxResponseBytes int64
}

// Client queries one SearXNG instance.
type Client struct {
	searchURL       *url.URL
	httpClient      HTTPClient
	userAgent       string
	maxResponseSize int64
}

// NewClient validates the instance URL and applies bounded defaults.
func NewClient(option ClientOption) (*Client, error) {
	searchURL, err := normalizeSearchURL(option.URL)
	if err != nil {
		return nil, err
	}
	httpClient := option.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	userAgent := strings.TrimSpace(option.UserAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	maxResponseSize := option.MaxResponseBytes
	if maxResponseSize <= 0 {
		maxResponseSize = defaultMaxResponseBytes
	}
	return &Client{
		searchURL:       searchURL,
		httpClient:      httpClient,
		userAgent:       userAgent,
		maxResponseSize: maxResponseSize,
	}, nil
}

func normalizeSearchURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("searchwire: URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("searchwire: parse URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("searchwire: URL must be an absolute HTTP(S) URL")
	}
	path := strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(path, "/search") && path != "search" {
		path += "/search"
	}
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	u.Path = path
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u, nil
}

// Search executes one JSON search request.
func (c *Client) Search(ctx context.Context, input SearchInput) (*SearchOutput, error) {
	if c == nil {
		return nil, fmt.Errorf("searchwire: nil client")
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, fmt.Errorf("searchwire: query is required")
	}
	if input.PageNumber != nil && *input.PageNumber == 0 {
		return nil, fmt.Errorf("searchwire: page number must be at least 1")
	}
	if input.SafeSearch != nil && (*input.SafeSearch < SafeSearchNone || *input.SafeSearch > SafeSearchStrict) {
		return nil, fmt.Errorf("searchwire: safe search must be 0, 1, or 2")
	}

	requestURL := *c.searchURL
	params := requestURL.Query()
	params.Set("q", query)
	params.Set("format", "json")
	setCommaSeparated(params, "categories", input.Categories)
	setCommaSeparated(params, "engines", input.Engines)
	if input.Language != nil {
		params.Set("language", *input.Language)
	}
	if input.PageNumber != nil {
		params.Set("pageno", strconv.FormatUint(uint64(*input.PageNumber), 10))
	}
	if input.TimeRange != nil {
		params.Set("time_range", string(*input.TimeRange))
	}
	if input.SafeSearch != nil {
		params.Set("safesearch", strconv.Itoa(int(*input.SafeSearch)))
	}
	requestURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("searchwire: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searchwire: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("searchwire: read response: %w", err)
	}
	if int64(len(body)) > c.maxResponseSize {
		return nil, ErrResponseTooLarge
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType != "application/json" && !strings.HasSuffix(mediaType, "+json") {
		return nil, fmt.Errorf("%w: %q", ErrUnexpectedContentType, mediaType)
	}
	var output SearchOutput
	if err := json.Unmarshal(body, &output); err != nil {
		return nil, fmt.Errorf("searchwire: decode response: %w", err)
	}
	return &output, nil
}

func setCommaSeparated[T ~string](params url.Values, key string, values []T) {
	if len(values) == 0 {
		return
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	params.Set(key, strings.Join(parts, ","))
}
