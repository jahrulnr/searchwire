# searchwire

Zero-config Go metasearch runtime for agent tooling.

Searchwire runs ordinary web/text searches for you. Provide a query only — no SearXNG deployment, search host, engine selection, or API key is required for the default path.

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
	searcher := searchwire.New()
	resp, err := searcher.Search(context.Background(), "Go context cancellation")
	if err != nil {
		log.Fatal(err)
	}
	for _, result := range resp.Results {
		fmt.Println(result.Title, result.URL)
	}
	for _, sourceErr := range resp.Errors {
		log.Printf("source %s failed: %s", sourceErr.Source, sourceErr.Error)
	}
}
```

## Built-in sources

Searchwire fans out concurrently to:

1. Brave Search (HTML)
2. Startpage (HTML)
3. Wikipedia MediaWiki API (JSON)

Source ordering is also the tie-break order when fused scores are equal.

## Partial failures

When at least one source succeeds, `Search` returns a `*Response` with merged results and any source failures in `Response.Errors`. When every source fails, `Search` returns a `*SearchError` listing each failure.

## Advanced options

```go
searcher := searchwire.New(
	searchwire.WithHTTPClient(httpClient),
	searchwire.WithLimit(5),
)
```

`WithHTTPClient(nil)` is ignored. `WithLimit` applies only to positive values.

## Limitations

- HTML adapters can break when source markup changes.
- Ordinary text/web results only; no images, news, maps, or shopping categories.
- No CAPTCHA solving, proxy rotation, or anti-bot bypasses.
- Not a promise of SearXNG parity or engine compatibility.

## Development

```bash
gofmt -w *.go
go mod tidy
go test ./...
go test -race ./...
go vet ./...
SEARCHWIRE_LIVE=1 go test -run TestLiveSearch -v
```

## Identity and license

The metasearch aggregation concept is inspired by systems such as [SearXNG](https://github.com/searxng/searxng), but Searchwire is **not** a SearXNG client, port, configuration-compatible implementation, or source-code derivative. SearXNG is AGPL; Searchwire is MIT.

This library is pre-v1; APIs may change.

MIT License. See [LICENSE](LICENSE).
