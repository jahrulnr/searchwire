package searchwire

import (
	"net/http"
	"os"
	"strings"
	"time"
)

// Config configures a Searcher. The zero value uses built-in defaults and
// zero-configuration sources. Optional integrations read credentials from
// explicit fields first, then from named environment variables.
type Config struct {
	HTTPClient       HTTPClient
	UserAgent        string
	Limit            int
	MaxResponseBytes int64
	Timeout          time.Duration

	GitHub GitHubConfig
	// Google and Custom are future-dev placeholders. They are not read by New()
	// until their source adapters are implemented.
	Google GoogleConfig
	Custom CustomSearchConfig
}

// GitHubConfig controls the built-in GitHub Search API source.
type GitHubConfig struct {
	// Enabled registers the GitHub source. Default true.
	Enabled *bool

	// Token is sent as Bearer auth. When empty, TokenEnv is read.
	Token string
	// TokenEnv names the environment variable for Token (default GITHUB_TOKEN).
	TokenEnv string

	// SearchIssues enables repository+issues mode. When false, only repositories
	// are searched. Default true. When true, repository search still serves as
	// the fallback if issues search fails.
	SearchIssues *bool
}

// GoogleConfig is a future-dev placeholder for Google Programmable Search Engine.
// Setting these fields has no effect on Search today.
type GoogleConfig struct {
	// APIKey is the Google Custom Search JSON API key (future).
	APIKey string
	// APIKeyEnv names the env var for APIKey (planned default: GOOGLE_API_KEY).
	APIKeyEnv string
	// CX is the programmable search engine ID (future).
	CX string
	// CXEnv names the env var for CX (planned default: GOOGLE_CX).
	CXEnv string
}

// CustomSearchConfig is a future-dev placeholder for a caller-provided search
// endpoint. Setting URL has no effect on Search today.
type CustomSearchConfig struct {
	URL string
}

// DefaultConfig returns the zero-configuration defaults.
func DefaultConfig() Config {
	return Config{}
}

func (c Config) withDefaults() Config {
	if c.HTTPClient == nil {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		c.HTTPClient = &http.Client{Timeout: timeout}
	}
	if strings.TrimSpace(c.UserAgent) == "" {
		c.UserAgent = defaultUserAgent
	}
	if c.Limit <= 0 {
		c.Limit = defaultLimit
	}
	if c.MaxResponseBytes <= 0 {
		c.MaxResponseBytes = defaultMaxResponseBytes
	}
	return c
}

type githubSettings struct {
	enabled      bool
	token        string
	searchIssues bool
}

func (g GitHubConfig) resolved() githubSettings {
	settings := githubSettings{
		enabled:      true,
		searchIssues: true,
	}
	if g.Enabled != nil {
		settings.enabled = *g.Enabled
	}
	if g.SearchIssues != nil {
		settings.searchIssues = *g.SearchIssues
	}
	settings.token = strings.TrimSpace(g.Token)
	if settings.token == "" {
		env := strings.TrimSpace(g.TokenEnv)
		if env == "" {
			env = "GITHUB_TOKEN"
		}
		settings.token = strings.TrimSpace(os.Getenv(env))
	}
	return settings
}

func sourcesFromConfig(cfg Config) []source {
	sources := []source{
		braveSource{},
		startpageSource{},
		wikipediaSource{},
	}
	if cfg.GitHub.resolved().enabled {
		settings := cfg.GitHub.resolved()
		sources = append(sources, githubSource{
			token:        settings.token,
			searchIssues: settings.searchIssues,
		})
	}
	return sources
}
