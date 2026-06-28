package searchwire

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type braveSource struct{}

func (braveSource) name() string { return "brave" }

func (braveSource) search(ctx context.Context, f *fetcher, query string, limit int) ([]sourceResult, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	endpoint := url.URL{
		Scheme: "https",
		Host:   "search.brave.com",
		Path:   "/search",
	}
	params := endpoint.Query()
	params.Set("q", query)
	params.Set("source", "web")
	endpoint.RawQuery = params.Encode()

	body, err := f.get(ctx, endpoint.String(), "text/html", []string{"text/html"})
	if err != nil {
		return nil, err
	}
	return parseBraveResults(body, limit)
}

func parseBraveResults(body []byte, limit int) ([]sourceResult, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse brave html: %w", err)
	}
	results := make([]sourceResult, 0, limit)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			if attrValue(n, "data-type") == "web" && hasClassToken(attrValue(n, "class"), "snippet") {
				if item, ok := parseBraveBlock(n); ok {
					item.rank = len(results) + 1
					results = append(results, item)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return results, nil
}

func parseBraveBlock(root *html.Node) (sourceResult, bool) {
	linkNode := findDescendantAnchor(root)
	if linkNode == nil {
		return sourceResult{}, false
	}
	resultURL := strings.TrimSpace(attrValue(linkNode, "href"))
	if !isSafeResultURL(resultURL) {
		return sourceResult{}, false
	}

	titleNode := findDescendant(root, "div", "search-snippet-title")
	if titleNode == nil {
		titleNode = findDescendant(root, "div", "title")
	}
	title := textContent(titleNode)
	if title == "" {
		title = textContent(linkNode)
	}
	if title == "" {
		return sourceResult{}, false
	}

	snippetNode := findDescendant(root, "div", "generic-snippet")
	snippet := ""
	if snippetNode != nil {
		if content := snippetNode.FirstChild; content != nil {
			snippet = textContent(content)
		}
		if snippet == "" {
			snippet = textContent(snippetNode)
		}
	}

	return sourceResult{title: title, url: resultURL, snippet: snippet}, true
}
