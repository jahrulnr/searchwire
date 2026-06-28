package searchwire

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGitHubConfigFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test_token")
	settings := GitHubConfig{TokenEnv: "GITHUB_TOKEN"}.resolved()
	if settings.token != "ghp_test_token" {
		t.Fatalf("token = %q", settings.token)
	}
}

func TestGitHubConfigExplicitTokenOverridesEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "from-env")
	settings := GitHubConfig{Token: "from-config"}.resolved()
	if settings.token != "from-config" {
		t.Fatalf("token = %q", settings.token)
	}
}

func TestGitHubDisabledRemovesSource(t *testing.T) {
	disabled := false
	s := New(Config{GitHub: GitHubConfig{Enabled: &disabled}})
	if len(s.sources) != 3 {
		t.Fatalf("sources = %d", len(s.sources))
	}
}

func TestGitHubReposOnlyMode(t *testing.T) {
	issues := false
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/search/issues") {
			t.Fatal("issues endpoint should not be called in repos-only mode")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(readFixtureString(t, "github-repos.json"))),
		}, nil
	})}
	s := New(Config{
		HTTPClient: client,
		GitHub:     GitHubConfig{SearchIssues: &issues},
	})
	if len(s.sources) != 4 {
		t.Fatalf("sources = %d", len(s.sources))
	}
	if _, err := s.Search(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
}

func TestGitHubTokenSentAsBearer(t *testing.T) {
	var auth string
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.Host, "api.github.com") {
			auth = r.Header.Get("Authorization")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"items":[]}`)),
		}, nil
	})}
	issues := false
	s := newSearcherWithSources([]source{githubSource{
		token:        "ghp_example",
		searchIssues: issues,
	}}, Config{HTTPClient: client})
	if _, err := s.Search(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer ghp_example" {
		t.Fatalf("authorization = %q", auth)
	}
}

func TestDefaultConfigZeroValue(t *testing.T) {
	if got := DefaultConfig(); got.Limit != 0 || got.GitHub.Token != "" {
		t.Fatalf("default config = %#v", got)
	}
	cfg := DefaultConfig().withDefaults()
	if cfg.Limit != defaultLimit {
		t.Fatalf("limit = %d", cfg.Limit)
	}
}
