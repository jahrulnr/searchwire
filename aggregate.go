package searchwire

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type sourceOutcome struct {
	index   int
	name    string
	results []sourceResult
	err     error
}

type fusedCandidate struct {
	title          string
	url            string
	snippet        string
	sources        []string
	score          float64
	sourcePriority int
	bestRank       int
}

var trackingParamPrefixes = []string{"utm_", "fbclid", "gclid"}

func fuseResults(outcomes []sourceOutcome, sources []source, limit int) []Result {
	byKey := make(map[string]*fusedCandidate)
	sourcePriority := make(map[string]int, len(sources))
	for i, src := range sources {
		sourcePriority[src.name()] = i
	}

	for _, outcome := range outcomes {
		srcName := outcome.name
		priority := sourcePriority[srcName]
		for _, item := range outcome.results {
			key, canonical, err := canonicalURL(item.url)
			if err != nil || key == "" {
				continue
			}
			existing, ok := byKey[key]
			score := 1.0 / (rrfK + float64(item.rank))
			if !ok {
				byKey[key] = &fusedCandidate{
					title:          item.title,
					url:            canonical,
					snippet:        item.snippet,
					sources:        []string{srcName},
					score:          score,
					sourcePriority: priority,
					bestRank:       item.rank,
				}
				continue
			}
			existing.score += score
			if !containsString(existing.sources, srcName) {
				existing.sources = appendSourceInOrder(existing.sources, srcName, sources)
			}
			if len(item.title) > len(existing.title) {
				existing.title = item.title
			}
			if len(item.snippet) > len(existing.snippet) {
				existing.snippet = item.snippet
			}
			if priority < existing.sourcePriority || (priority == existing.sourcePriority && item.rank < existing.bestRank) {
				existing.sourcePriority = priority
				existing.bestRank = item.rank
			}
		}
	}

	candidates := make([]*fusedCandidate, 0, len(byKey))
	for _, candidate := range byKey {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if a.sourcePriority != b.sourcePriority {
			return a.sourcePriority < b.sourcePriority
		}
		if a.bestRank != b.bestRank {
			return a.bestRank < b.bestRank
		}
		return a.url < b.url
	})

	if limit > len(candidates) {
		limit = len(candidates)
	}
	results := make([]Result, limit)
	for i := 0; i < limit; i++ {
		c := candidates[i]
		results[i] = Result{
			Title:   c.title,
			URL:     c.url,
			Snippet: c.snippet,
			Sources: append([]string(nil), c.sources...),
			Score:   c.score,
		}
	}
	return results
}

func appendSourceInOrder(existing []string, name string, sources []source) []string {
	if containsString(existing, name) {
		return existing
	}
	order := make(map[string]int, len(sources))
	for i, src := range sources {
		order[src.name()] = i
	}
	existing = append(existing, name)
	sort.Slice(existing, func(i, j int) bool {
		return order[existing[i]] < order[existing[j]]
	})
	return existing
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func canonicalURL(raw string) (key string, canonical string, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", fmt.Errorf("unsupported scheme")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""

	if u.Scheme == "http" && strings.HasSuffix(u.Host, ":80") {
		u.Host = strings.TrimSuffix(u.Host, ":80")
	}
	if u.Scheme == "https" && strings.HasSuffix(u.Host, ":443") {
		u.Host = strings.TrimSuffix(u.Host, ":443")
	}

	if u.Path == "" {
		u.Path = "/"
	} else if u.Path != "/" && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	if u.RawQuery != "" {
		values, parseErr := url.ParseQuery(u.RawQuery)
		if parseErr == nil {
			for keyName := range values {
				lower := strings.ToLower(keyName)
				for _, prefix := range trackingParamPrefixes {
					if lower == prefix || strings.HasPrefix(lower, prefix) {
						values.Del(keyName)
						break
					}
				}
			}
			u.RawQuery = values.Encode()
		}
	}

	canonical = u.String()
	return canonical, canonical, nil
}
