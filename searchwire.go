package searchwire

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const (
	defaultLimit            = 10
	defaultTimeout          = 10 * time.Second
	defaultMaxResponseBytes = 2 << 20
	defaultUserAgent        = "searchwire/0 (+https://github.com/jahrulnr/searchwire)"
)

// HTTPClient is satisfied by *http.Client and lightweight test doubles.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Option configures a Searcher.
type Option func(*Searcher)

// WithHTTPClient sets the HTTP client used for all source requests.
// Nil is ignored.
func WithHTTPClient(client HTTPClient) Option {
	return func(s *Searcher) {
		if client != nil {
			s.httpClient = client
		}
	}
}

// WithLimit sets the maximum number of merged results returned.
// Non-positive values are ignored.
func WithLimit(limit int) Option {
	return func(s *Searcher) {
		if limit > 0 {
			s.limit = limit
		}
	}
}

// Searcher fans out to built-in sources, merges duplicates, and ranks results.
type Searcher struct {
	httpClient      HTTPClient
	userAgent       string
	limit           int
	maxResponseSize int64
	sources         []source
}

// New returns a Searcher with built-in sources and sensible defaults.
func New(options ...Option) *Searcher {
	s := &Searcher{
		httpClient:      &http.Client{Timeout: defaultTimeout},
		userAgent:       defaultUserAgent,
		limit:           defaultLimit,
		maxResponseSize: defaultMaxResponseBytes,
		sources:         defaultSources(),
	}
	for _, opt := range options {
		opt(s)
	}
	return s
}

// Response is the merged output of a metasearch query.
type Response struct {
	Query   string
	Results []Result
	Errors  []SourceError
}

// Result is one ranked web/text hit after deduplication and fusion.
type Result struct {
	Title   string
	URL     string
	Snippet string
	Sources []string
	Score   float64
}

// Search runs the query across all built-in sources concurrently.
// When at least one source succeeds, partial failures are returned in Response.Errors.
func (s *Searcher) Search(ctx context.Context, query string) (*Response, error) {
	if s == nil {
		return nil, ErrNilSearcher
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ch := make(chan sourceOutcome, len(s.sources))
	for i, src := range s.sources {
		go func(index int, src source) {
			fetcher := newFetcher(s.httpClient, s.userAgent, s.maxResponseSize)
			results, err := src.search(ctx, fetcher, query, s.limit)
			ch <- sourceOutcome{index: index, name: src.name(), results: results, err: err}
		}(i, src)
	}

	successes := make([]sourceOutcome, 0, len(s.sources))
	var failures []SourceError
	for range s.sources {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case outcome := <-ch:
			if outcome.err != nil {
				failures = append(failures, SourceError{Source: outcome.name, Error: outcome.err.Error()})
				continue
			}
			successes = append(successes, outcome)
		}
	}

	if len(successes) == 0 {
		return nil, &SearchError{Failures: orderSourceErrors(failures, s.sources)}
	}

	results := fuseResults(successes, s.sources, s.limit)
	return &Response{Query: query, Results: results, Errors: orderSourceErrors(failures, s.sources)}, nil
}
