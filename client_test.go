package searchwire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestNewClientValidatesAndNormalizesURL(t *testing.T) {
	for _, raw := range []string{"", "relative", "ftp://example.com"} {
		if _, err := NewClient(ClientOption{URL: raw}); err == nil {
			t.Errorf("NewClient(%q) accepted an invalid URL", raw)
		}
	}
	client, err := NewClient(ClientOption{URL: "https://example.com/searx/"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := client.searchURL.String(), "https://example.com/searx/search"; got != want {
		t.Fatalf("search URL = %q, want %q", got, want)
	}
	client, err = NewClient(ClientOption{URL: "https://example.com/search"})
	if err != nil {
		t.Fatal(err)
	}
	if got := client.searchURL.Path; got != "/search" {
		t.Fatalf("search path = %q", got)
	}
}

func TestSearchSendsStableParameters(t *testing.T) {
	var request *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"query":"Tokyo","results":[]}`)
	}))
	defer server.Close()

	language := "id-ID"
	page := uint(2)
	timeRange := TimeRangeMonth
	safeSearch := SafeSearchStrict
	client, err := NewClient(ClientOption{URL: server.URL, UserAgent: "searchwire-test/1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(context.Background(), SearchInput{
		Query:      "  Tokyo  ",
		Categories: []Category{"general", "it"},
		Engines:    []SearchEngine{"startpage", "github"},
		Language:   &language,
		PageNumber: &page,
		TimeRange:  &timeRange,
		SafeSearch: &safeSearch,
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodGet || request.URL.Path != "/search" {
		t.Fatalf("request = %s %s", request.Method, request.URL.Path)
	}
	want := url.Values{
		"q":          {"Tokyo"},
		"format":     {"json"},
		"categories": {"general,it"},
		"engines":    {"startpage,github"},
		"language":   {"id-ID"},
		"pageno":     {"2"},
		"time_range": {"month"},
		"safesearch": {"2"},
	}
	if got := request.URL.Query(); got.Encode() != want.Encode() {
		t.Fatalf("query = %q, want %q", got.Encode(), want.Encode())
	}
	if got := request.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("Accept = %q", got)
	}
	if got := request.Header.Get("User-Agent"); got != "searchwire-test/1" {
		t.Fatalf("User-Agent = %q", got)
	}
}

func TestSearchDecodesCurrentAndUnknownFields(t *testing.T) {
	fixture, err := os.ReadFile("testdata/current-response.json")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	client, err := NewClient(ClientOption{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	output, err := client.Search(context.Background(), SearchInput{Query: "Tokyo"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Query != "Tokyo" || len(output.Results) != 2 {
		t.Fatalf("unexpected output: query=%q results=%d", output.Query, len(output.Results))
	}
	if output.Results[1].Category != "packages" {
		t.Fatalf("unknown category was not preserved: %#v", output.Results[1])
	}
	if len(output.Answers) != 2 || output.Answers[0].Text == "" || output.Answers[1].Text != "Legacy answer" {
		t.Fatalf("answers = %#v", output.Answers)
	}
	if len(output.Infoboxes) != 1 || len(output.Infoboxes[0].URLs) != 1 {
		t.Fatalf("infoboxes = %#v", output.Infoboxes)
	}
	if len(output.UnresponsiveEngines) != 1 {
		t.Fatalf("unresponsive engines = %#v", output.UnresponsiveEngines)
	}
}

func TestSearchRejectsInvalidInputBeforeHTTP(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	client, err := NewClient(ClientOption{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	page := uint(0)
	badSafeSearch := SafeSearchLevel(3)
	for name, input := range map[string]SearchInput{
		"empty query":     {},
		"zero page":       {Query: "x", PageNumber: &page},
		"bad safe search": {Query: "x", SafeSearch: &badSafeSearch},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := client.Search(context.Background(), input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	if called {
		t.Fatal("invalid input reached HTTP server")
	}
	var nilClient *Client
	if _, err := nilClient.Search(context.Background(), SearchInput{Query: "x"}); err == nil {
		t.Fatal("nil client did not return an error")
	}
}

func TestSearchHandlesHTTPAndResponseFailures(t *testing.T) {
	t.Run("HTTP status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
		}))
		defer server.Close()
		client, _ := NewClient(ClientOption{URL: server.URL})
		_, err := client.Search(context.Background(), SearchInput{Query: "x"})
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("HTML success response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html>not JSON</html>")
		}))
		defer server.Close()
		client, _ := NewClient(ClientOption{URL: server.URL})
		_, err := client.Search(context.Background(), SearchInput{Query: "x"})
		if !errors.Is(err, ErrUnexpectedContentType) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("oversized response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, strings.Repeat("x", 17))
		}))
		defer server.Close()
		client, _ := NewClient(ClientOption{URL: server.URL, MaxResponseBytes: 16})
		_, err := client.Search(context.Background(), SearchInput{Query: "x"})
		if !errors.Is(err, ErrResponseTooLarge) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "not-json")
		}))
		defer server.Close()
		client, _ := NewClient(ClientOption{URL: server.URL})
		if _, err := client.Search(context.Background(), SearchInput{Query: "x"}); err == nil {
			t.Fatal("expected decoding error")
		}
	})
}

func TestSearchPropagatesCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client, _ := NewClient(ClientOption{URL: server.URL})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Search(ctx, SearchInput{Query: "x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}
