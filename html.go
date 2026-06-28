package searchwire

import (
	"strings"

	"golang.org/x/net/html"
)

func attrValue(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func hasClassToken(classValue, token string) bool {
	for _, part := range strings.Fields(classValue) {
		if part == token {
			return true
		}
	}
	return strings.Contains(classValue, token)
}

func textContent(node *html.Node) string {
	if node == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return collapseWhitespace(b.String())
}

func collapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func findDescendant(node *html.Node, tag string, classContains string) *html.Node {
	if node == nil {
		return nil
	}
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode {
			if tag == "" || n.Data == tag {
				if classContains == "" || hasClassToken(attrValue(n, "class"), classContains) {
					found = n
					return
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return found
}

func findDescendantAnchor(node *html.Node) *html.Node {
	if node == nil {
		return nil
	}
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			if href := attrValue(n, "href"); isSafeResultURL(href) {
				found = n
				return
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return found
}
