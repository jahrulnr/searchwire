package searchwire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseStartpageFixture(t *testing.T) {
	results, err := parseStartpageResults(readFixture(t, "startpage-results.html"), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	if results[0].url != "https://go.dev/" || results[1].url != "https://pkg.go.dev/" {
		t.Fatalf("results = %#v", results)
	}
}

func TestStartpageSearchRequestContract(t *testing.T) {
	var req *http.Request
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		req = r
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
			Body:       io.NopCloser(strings.NewReader(readFixtureString(t, "startpage-results.html"))),
		}, nil
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	if _, err := (startpageSource{}).search(context.Background(), f, "Go", 5); err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodGet || !strings.Contains(req.URL.String(), "www.startpage.com/sp/search") {
		t.Fatalf("url = %s", req.URL)
	}
	if req.URL.Query().Get("query") != "Go" {
		t.Fatalf("query = %s", req.URL.RawQuery)
	}
}

func TestStartpageRejectsNon2xx(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Body: io.NopCloser(strings.NewReader("blocked"))}, nil
	})}
	f := newFetcher(client, defaultUserAgent, defaultMaxResponseBytes)
	_, err := (startpageSource{}).search(context.Background(), f, "Go", 5)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
		t.Fatalf("err = %v", err)
	}
}

func TestStartpageMalformedHTML(t *testing.T) {
	results, err := parseStartpageResults([]byte("<html><div class=\"result\">broken</div>"), 10)
	if err != nil || len(results) != 0 {
		t.Fatalf("results = %#v err = %v", results, err)
	}
}
