package searchwire

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type startpageSource struct{}

func (startpageSource) name() string { return "startpage" }

func (startpageSource) search(ctx context.Context, f *fetcher, query string, limit int) ([]sourceResult, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	endpoint := url.URL{
		Scheme: "https",
		Host:   "www.startpage.com",
		Path:   "/sp/search",
	}
	params := endpoint.Query()
	params.Set("query", query)
	endpoint.RawQuery = params.Encode()

	body, err := f.get(ctx, endpoint.String(), "text/html", []string{"text/html"})
	if err != nil {
		return nil, err
	}
	return parseStartpageResults(body, limit)
}

func parseStartpageResults(body []byte, limit int) ([]sourceResult, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse startpage html: %w", err)
	}
	results := make([]sourceResult, 0, limit)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" && hasClassToken(attrValue(n, "class"), "result") {
			if item, ok := parseStartpageBlock(n); ok {
				item.rank = len(results) + 1
				results = append(results, item)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return results, nil
}

func parseStartpageBlock(root *html.Node) (sourceResult, bool) {
	linkNode := findStartpagePrimaryLink(root)
	if linkNode == nil {
		return sourceResult{}, false
	}
	resultURL := strings.TrimSpace(attrValue(linkNode, "href"))
	if !isSafeResultURL(resultURL) {
		return sourceResult{}, false
	}

	titleNode := findDescendant(root, "h2", "wgl-title")
	title := textContent(titleNode)
	if title == "" {
		title = textContent(linkNode)
	}
	if title == "" {
		return sourceResult{}, false
	}

	snippetNode := findDescendant(root, "p", "description")
	snippet := textContent(snippetNode)

	return sourceResult{title: title, url: resultURL, snippet: snippet}, true
}

func findStartpagePrimaryLink(root *html.Node) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			classValue := attrValue(n, "class")
			if hasClassToken(classValue, "result-title") || hasClassToken(classValue, "result-link") {
				if href := attrValue(n, "href"); isSafeResultURL(href) {
					found = n
					return
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return found
}
