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
	searcher := searchwire.New(searchwire.Config{})
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

`searchwire.New(searchwire.Config{})` and `searchwire.New(searchwire.DefaultConfig())` are equivalent.

## Built-in sources

Searchwire fans out concurrently to:

1. Brave Search (HTML)
2. Startpage (HTML)
3. Wikipedia MediaWiki API (JSON)
4. GitHub Search API (repositories + issues; falls back to repositories when issues search fails)

Source ordering is also the tie-break order when fused scores are equal.

Unauthenticated GitHub search is limited to about 10 requests per minute across search endpoints.

## Configuration

`searchwire.Config` groups runtime settings and optional integrations:

```go
searcher := searchwire.New(searchwire.Config{
	Limit:      5,
	HTTPClient: httpClient,
	GitHub: searchwire.GitHubConfig{
		Token: "ghp_...", // or read GITHUB_TOKEN
	},
})
```

### GitHub

| Field | Default | Description |
|-------|---------|-------------|
| `Enabled` | `true` | Register the GitHub source |
| `Token` | — | Bearer token; overrides env |
| `TokenEnv` | `GITHUB_TOKEN` | Env var for token |
| `SearchIssues` | `true` | Repo+issues (B); `false` = repos only (A) |

When `SearchIssues` is true and issues search fails, repository results are still returned.

### Future development

These `Config` fields exist for upcoming work only. **They are ignored by `New()` today.**

| Block | Planned integration |
|-------|---------------------|
| `GoogleConfig` | Google Programmable Search JSON API (`APIKey` / `GOOGLE_API_KEY`, `CX` / `GOOGLE_CX`) |
| `CustomSearchConfig` | Caller-provided search endpoint (`URL`) |

Do not rely on them for behavior until a source adapter lands and tests cover it.

## Partial failures

When at least one source succeeds, `Search` returns a `*Response` with merged results and any source failures in `Response.Errors`. When every source fails, `Search` returns a `*SearchError` listing each failure.

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
