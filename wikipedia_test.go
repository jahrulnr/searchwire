package searchwire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseWikipediaFixture(t *testing.T) {
	results, err := parseWikipediaResults(readFixture(t, "wikipedia-results.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	if results[0].title != "Go" || results[1].title != "Go (programming language)" {
		t.Fatalf("order = %#v", results)
	}
}

func TestWikipediaMissingQuery(t *testing.T) {
	body := []byte(`{"batchcomplete":""}`)
	results, err := parseWikipediaResults(body)
	if err != nil || len(results) != 0 {
		t.Fatalf("results = %#v err = %v", results, err)
	}
}

func TestWikipediaMissingExtract(t *testing.T) {
	body := []byte(`{"query":{"pages":[{"pageid":1,"title":"Go","index":1,"fullurl":"https://en.wikipedia.org/wiki/Go"}]}}`)
	results, err := parseWikipediaResults(body)
	if err != nil || len(results) != 1 || results[0].snippet != "" {
		t.Fatalf("results = %#v err = %v", results, err)
	}
}

func TestWikipediaSearchRequestContract(t *testing.T) {
	var req *http.Request
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		req = r
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(readFixtureString(t, "wikipedia-results.json"))),
		}, nil
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	if _, err := (wikipediaSource{}).search(context.Background(), f, "Go", 5); err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodGet || !strings.Contains(req.URL.String(), "en.wikipedia.org/w/api.php") {
		t.Fatalf("url = %s", req.URL)
	}
	q := req.URL.Query()
	if q.Get("action") != "query" || q.Get("generator") != "search" || q.Get("gsrsearch") != "Go" {
		t.Fatalf("query = %s", req.URL.RawQuery)
	}
}

func TestWikipediaMalformedJSON(t *testing.T) {
	_, err := parseWikipediaResults([]byte(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWikipediaRequestContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	_, err := (wikipediaSource{}).search(ctx, f, "Go", 5)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
