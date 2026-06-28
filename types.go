package searchwire

import (
	"bytes"
	"encoding/json"
)

// Category names a search category configured by a SearXNG instance.
type Category string

// SearchEngine names a search engine configured by a SearXNG instance.
type SearchEngine string

// TimeRange limits results to a backend-supported time window.
type TimeRange string

// SafeSearchLevel controls backend-supported result filtering.
type SafeSearchLevel int

const (
	TimeRangeDay   TimeRange = "day"
	TimeRangeWeek  TimeRange = "week"
	TimeRangeMonth TimeRange = "month"
	TimeRangeYear  TimeRange = "year"

	SafeSearchNone     SafeSearchLevel = 0
	SafeSearchModerate SafeSearchLevel = 1
	SafeSearchStrict   SafeSearchLevel = 2
)

// SearchInput contains the stable subset of SearXNG search parameters.
// Query is required; all other fields are optional.
type SearchInput struct {
	Query      string
	Categories []Category
	Engines    []SearchEngine
	Language   *string
	PageNumber *uint
	TimeRange  *TimeRange
	SafeSearch *SafeSearchLevel
}

// SearchOutput models the useful, stable JSON response envelope. Unknown
// fields are deliberately ignored so newer SearXNG releases remain usable.
type SearchOutput struct {
	Query               string     `json:"query"`
	Results             []Result   `json:"results"`
	Answers             []Answer   `json:"answers"`
	Corrections         []string   `json:"corrections"`
	Infoboxes           []Infobox  `json:"infoboxes"`
	Suggestions         []string   `json:"suggestions"`
	UnresponsiveEngines [][]string `json:"unresponsive_engines"`
}

// Result intentionally contains only fields shared by ordinary web results.
// Category and engine names are strings because they are instance-defined.
type Result struct {
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Content       string   `json:"content"`
	Engine        string   `json:"engine"`
	Engines       []string `json:"engines"`
	Category      string   `json:"category"`
	Template      string   `json:"template"`
	PublishedDate *string  `json:"publishedDate"`
	Thumbnail     string   `json:"thumbnail"`
	ImageSource   string   `json:"img_src"`
}

// Answer supports both modern object answers and the legacy string form.
type Answer struct {
	Text     string `json:"answer"`
	URL      string `json:"url"`
	Engine   string `json:"engine"`
	Template string `json:"template"`
}

func (a *Answer) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*a = Answer{}
		return nil
	}
	var legacy string
	if err := json.Unmarshal(data, &legacy); err == nil {
		*a = Answer{Text: legacy}
		return nil
	}
	type answer Answer
	var decoded answer
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*a = Answer(decoded)
	return nil
}

// Infobox contains a knowledge-panel style response and its source links.
type Infobox struct {
	Title       string `json:"infobox"`
	ID          string `json:"id"`
	Content     string `json:"content"`
	Engine      string `json:"engine"`
	ImageSource string `json:"img_src"`
	URLs        []Link `json:"urls"`
}

// Link is a titled URL included in an infobox.
type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
