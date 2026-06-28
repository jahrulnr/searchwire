package searchwire

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLiveSearch(t *testing.T) {
	if os.Getenv("SEARCHWIRE_LIVE") != "1" {
		t.Skip("set SEARCHWIRE_LIVE=1 to run live network smoke test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := New(Config{}).Search(ctx, "Go programming language")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected at least one merged result")
	}
	validURL := false
	for _, result := range resp.Results {
		if strings.HasPrefix(result.URL, "http://") || strings.HasPrefix(result.URL, "https://") {
			validURL = true
			break
		}
	}
	if !validURL {
		t.Fatal("expected at least one http(s) result URL")
	}
	if len(resp.Errors) > 0 {
		for _, sourceErr := range resp.Errors {
			log.Printf("partial source failure: %s: %s", sourceErr.Source, sourceErr.Error)
		}
	}
}
