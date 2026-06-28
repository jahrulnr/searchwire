package searchwire

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveSearch(t *testing.T) {
	endpoint := os.Getenv("SEARXNG_URL")
	if endpoint == "" {
		t.Skip("SEARXNG_URL is not set")
	}
	client, err := NewClient(ClientOption{URL: endpoint, UserAgent: "searchwire-live-test/1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := client.Search(ctx, SearchInput{Query: "Tokyo"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Query != "Tokyo" {
		t.Fatalf("query = %q", output.Query)
	}
	if len(output.Results) == 0 && len(output.Answers) == 0 && len(output.Infoboxes) == 0 {
		t.Fatal("search returned no usable results")
	}
}
