package searchwire

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type githubSource struct {
	token        string
	searchIssues bool
}

func (g githubSource) name() string { return "github" }

func (g githubSource) search(ctx context.Context, f *fetcher, query string, limit int) ([]sourceResult, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if !g.searchIssues {
		return searchGitHubRepos(ctx, f, g.token, query, limit)
	}

	type outcome struct {
		results []sourceResult
		err     error
	}

	repoCh := make(chan outcome, 1)
	issueCh := make(chan outcome, 1)
	go func() {
		results, err := searchGitHubRepos(ctx, f, g.token, query, limit)
		repoCh <- outcome{results: results, err: err}
	}()
	go func() {
		results, err := searchGitHubIssues(ctx, f, g.token, query, limit)
		issueCh <- outcome{results: results, err: err}
	}()

	repoOut := <-repoCh
	issueOut := <-issueCh

	if repoOut.err != nil && issueOut.err != nil {
		return nil, fmt.Errorf("repos: %v; issues: %v", repoOut.err, issueOut.err)
	}
	if issueOut.err != nil {
		return repoOut.results, nil
	}
	if repoOut.err != nil {
		return nil, repoOut.err
	}
	return mergeGitHubResults(repoOut.results, issueOut.results, limit), nil
}

func searchGitHubRepos(ctx context.Context, f *fetcher, token, query string, limit int) ([]sourceResult, error) {
	endpoint := url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   "/search/repositories",
	}
	params := endpoint.Query()
	params.Set("q", query)
	params.Set("per_page", fmt.Sprintf("%d", limit))
	endpoint.RawQuery = params.Encode()

	body, err := f.getGitHub(ctx, endpoint.String(), token)
	if err != nil {
		return nil, err
	}
	return parseGitHubRepoResults(body)
}

func searchGitHubIssues(ctx context.Context, f *fetcher, token, query string, limit int) ([]sourceResult, error) {
	endpoint := url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   "/search/issues",
	}
	params := endpoint.Query()
	params.Set("q", query)
	params.Set("per_page", fmt.Sprintf("%d", limit))
	params.Set("sort", "relevance")
	endpoint.RawQuery = params.Encode()

	body, err := f.getGitHub(ctx, endpoint.String(), token)
	if err != nil {
		return nil, err
	}
	return parseGitHubIssueResults(body)
}

type githubSearchResponse struct {
	Items []json.RawMessage `json:"items"`
}

type githubRepoItem struct {
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
}

type githubIssueItem struct {
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func parseGitHubRepoResults(body []byte) ([]sourceResult, error) {
	var payload githubSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode github repo response: %w", err)
	}
	results := make([]sourceResult, 0, len(payload.Items))
	rank := 0
	for _, raw := range payload.Items {
		var item githubRepoItem
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		title := collapseWhitespace(item.FullName)
		pageURL := strings.TrimSpace(item.HTMLURL)
		if title == "" || !isSafeResultURL(pageURL) {
			continue
		}
		rank++
		results = append(results, sourceResult{
			title:   title,
			url:     pageURL,
			snippet: collapseWhitespace(item.Description),
			rank:    rank,
		})
	}
	return results, nil
}

func parseGitHubIssueResults(body []byte) ([]sourceResult, error) {
	var payload githubSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode github issue response: %w", err)
	}
	results := make([]sourceResult, 0, len(payload.Items))
	rank := 0
	for _, raw := range payload.Items {
		var item githubIssueItem
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		title := collapseWhitespace(item.Title)
		pageURL := strings.TrimSpace(item.HTMLURL)
		if title == "" || !isSafeResultURL(pageURL) {
			continue
		}
		rank++
		results = append(results, sourceResult{
			title:   title,
			url:     pageURL,
			snippet: truncateSnippet(collapseWhitespace(item.Body), 320),
			rank:    rank,
		})
	}
	return results, nil
}

func mergeGitHubResults(repos, issues []sourceResult, limit int) []sourceResult {
	seen := make(map[string]struct{})
	merged := make([]sourceResult, 0, limit)
	appendUnique := func(item sourceResult) {
		if len(merged) >= limit {
			return
		}
		key, _, err := canonicalURL(item.url)
		if err != nil {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		merged = append(merged, item)
	}

	i, j := 0, 0
	for len(merged) < limit && (i < len(repos) || j < len(issues)) {
		if i < len(repos) {
			appendUnique(repos[i])
			i++
		}
		if len(merged) >= limit {
			break
		}
		if j < len(issues) {
			appendUnique(issues[j])
			j++
		}
	}
	for idx := range merged {
		merged[idx].rank = idx + 1
	}
	return merged
}

func truncateSnippet(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max-3]) + "..."
}
