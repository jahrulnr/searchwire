package searchwire

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type wikipediaSource struct{}

func (wikipediaSource) name() string { return "wikipedia" }

func (wikipediaSource) search(ctx context.Context, f *fetcher, query string, limit int) ([]sourceResult, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	endpoint := url.URL{
		Scheme: "https",
		Host:   "en.wikipedia.org",
		Path:   "/w/api.php",
	}
	params := endpoint.Query()
	params.Set("action", "query")
	params.Set("generator", "search")
	params.Set("gsrsearch", query)
	params.Set("gsrlimit", fmt.Sprintf("%d", limit))
	params.Set("prop", "extracts|info")
	params.Set("exintro", "1")
	params.Set("explaintext", "1")
	params.Set("inprop", "url")
	params.Set("format", "json")
	params.Set("formatversion", "2")
	endpoint.RawQuery = params.Encode()

	body, err := f.get(ctx, endpoint.String(), "application/json", []string{"application/json"})
	if err != nil {
		return nil, err
	}
	return parseWikipediaResults(body)
}

type wikipediaResponse struct {
	Query struct {
		Pages []wikipediaPage `json:"pages"`
	} `json:"query"`
}

type wikipediaPage struct {
	PageID  int64  `json:"pageid"`
	Title   string `json:"title"`
	Index   int    `json:"index"`
	Extract string `json:"extract"`
	FullURL string `json:"fullurl"`
}

func parseWikipediaResults(body []byte) ([]sourceResult, error) {
	var payload wikipediaResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode wikipedia response: %w", err)
	}
	pages := append([]wikipediaPage(nil), payload.Query.Pages...)
	sort.SliceStable(pages, func(i, j int) bool {
		return pages[i].Index < pages[j].Index
	})

	results := make([]sourceResult, 0, len(pages))
	rank := 0
	for _, page := range pages {
		title := collapseWhitespace(page.Title)
		pageURL := strings.TrimSpace(page.FullURL)
		if title == "" || !isSafeResultURL(pageURL) {
			continue
		}
		rank++
		results = append(results, sourceResult{
			title:   title,
			url:     pageURL,
			snippet: collapseWhitespace(page.Extract),
			rank:    rank,
		})
	}
	return results, nil
}
