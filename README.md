# searchwire

`searchwire` is a small Go client for the subset of the SearXNG JSON Search API commonly needed by agent applications.

It deliberately models stable search primitives instead of mirroring every SearXNG engine, category, template, or Python result type. Unknown response fields and category names remain non-fatal, keeping the client useful as SearXNG evolves.

## Scope

Included:

- GET requests to a SearXNG `/search` endpoint.
- Query, category, engine, language, page, time-range, and safe-search parameters.
- Ordinary results, answers, corrections, infoboxes, suggestions, and unresponsive-engine metadata.
- Context cancellation, custom HTTP clients, bounded responses, and typed HTTP errors.
- Compatibility with legacy string answers.

Not included:

- Full parity with SearXNG's Python result-type hierarchy.
- A hardcoded catalog of engines or instance-specific categories.
- Public-instance discovery, failover, or health monitoring.
- Administration APIs, HTML scraping, or result rendering.

## Requirements

- Go 1.25 or newer.
- A SearXNG instance with the JSON response format enabled.

SearXNG instances commonly disable JSON responses by default. Configure the instance's `search.formats` list to include `json` before using it with this client.

## Install

```bash
go get github.com/jahrulnr/searchwire
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jahrulnr/searchwire"
)

func main() {
	client, err := searchwire.NewClient(searchwire.ClientOption{
		URL:       "https://search.example.com",
		UserAgent: "my-agent/1.0",
	})
	if err != nil {
		log.Fatal(err)
	}

	output, err := client.Search(context.Background(), searchwire.SearchInput{
		Query: "Go context cancellation",
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, result := range output.Results {
		fmt.Printf("%s\n%s\n", result.Title, result.URL)
	}
}
```

The client accepts either an instance root URL or an existing `/search` URL. For a subpath deployment such as `https://example.com/searx/`, it requests `https://example.com/searx/search`.

## Optional parameters

```go
language := "id-ID"
page := uint(2)
timeRange := searchwire.TimeRangeMonth
safeSearch := searchwire.SafeSearchStrict

output, err := client.Search(ctx, searchwire.SearchInput{
	Query:      "distributed systems",
	Categories: []searchwire.Category{"general", "it"},
	Engines:    []searchwire.SearchEngine{"startpage", "github"},
	Language:   &language,
	PageNumber: &page,
	TimeRange:  &timeRange,
	SafeSearch: &safeSearch,
})
```

Category and engine values are strings because each SearXNG instance controls which values are available.

## Errors and limits

- `HTTPError` contains the status code and a bounded response body for non-2xx responses.
- `ErrUnexpectedContentType` identifies instances that return HTML instead of JSON.
- `ErrResponseTooLarge` identifies responses over the configured limit, which defaults to 1 MiB.
- The default HTTP client timeout is 20 seconds. Supply `ClientOption.HTTPClient` to control transport or timeout behavior.

## Development

```bash
go test -race ./...
go vet ./...
```

Live SearXNG access is not required by the test suite; protocol behavior is tested with local HTTP fixtures.

## Stability

The API is pre-v1 and may change while its first production consumer is integrated. Scope expansion should be driven by demonstrated consumer needs rather than parity with SearXNG internals.

## License

[MIT](LICENSE)
