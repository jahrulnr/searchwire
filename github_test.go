package searchwire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseGitHubRepoFixture(t *testing.T) {
	results, err := parseGitHubRepoResults(readFixture(t, "github-repos.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].title != "golang/go" {
		t.Fatalf("results = %#v", results)
	}
}

func TestParseGitHubIssueFixture(t *testing.T) {
	results, err := parseGitHubIssueResults(readFixture(t, "github-issues.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].title != "Add context cancellation docs" {
		t.Fatalf("results = %#v", results)
	}
}

func TestMergeGitHubResultsInterleavesAndDedupes(t *testing.T) {
	repos := []sourceResult{
		{title: "golang/go", url: "https://github.com/golang/go", rank: 1},
		{title: "dup", url: "https://github.com/golang/go/", rank: 2},
	}
	issues := []sourceResult{
		{title: "issue", url: "https://github.com/golang/go/issues/1", rank: 1},
	}
	merged := mergeGitHubResults(repos, issues, 10)
	if len(merged) != 2 {
		t.Fatalf("merged = %#v", merged)
	}
	if merged[0].title != "golang/go" || merged[1].title != "issue" {
		t.Fatalf("order = %#v", merged)
	}
	if merged[0].rank != 1 || merged[1].rank != 2 {
		t.Fatalf("ranks = %#v", merged)
	}
}

func TestGitHubRepoRequestContract(t *testing.T) {
	var req *http.Request
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		req = r
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(readFixtureString(t, "github-repos.json"))),
		}, nil
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	if _, err := searchGitHubRepos(context.Background(), f, "", "go context", 5); err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodGet || !strings.Contains(req.URL.String(), "/search/repositories") {
		t.Fatalf("url = %s", req.URL)
	}
	if req.URL.Query().Get("q") != "go context" || req.Header.Get("Accept") != "application/vnd.github+json" {
		t.Fatalf("request = %s headers=%#v", req.URL, req.Header)
	}
}

func TestGitHubIssueRequestContract(t *testing.T) {
	var req *http.Request
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		req = r
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(readFixtureString(t, "github-issues.json"))),
		}, nil
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	if _, err := searchGitHubIssues(context.Background(), f, "", "go context", 5); err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodGet || !strings.Contains(req.URL.String(), "/search/issues") {
		t.Fatalf("url = %s", req.URL)
	}
}

func TestGitHubFallbackToReposWhenIssuesFail(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/search/issues") {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       io.NopCloser(strings.NewReader("rate limit")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(readFixtureString(t, "github-repos.json"))),
		}, nil
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	results, err := githubSource{searchIssues: true}.search(context.Background(), f, "go", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].title != "golang/go" {
		t.Fatalf("results = %#v", results)
	}
}

func TestGitHubFailsWhenReposFailEvenIfIssuesWork(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/search/repositories") {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       io.NopCloser(strings.NewReader("blocked")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(readFixtureString(t, "github-issues.json"))),
		}, nil
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	_, err := githubSource{searchIssues: true}.search(context.Background(), f, "go", 5)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
		t.Fatalf("err = %v", err)
	}
}

func TestGitHubMalformedJSON(t *testing.T) {
	_, err := parseGitHubRepoResults([]byte(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGitHubSkipsUnsafeURLs(t *testing.T) {
	body := []byte(`{"items":[{"title":"x","html_url":"javascript:alert(1)"}]}`)
	results, err := parseGitHubIssueResults(body)
	if err != nil || len(results) != 0 {
		t.Fatalf("results = %#v err = %v", results, err)
	}
}
