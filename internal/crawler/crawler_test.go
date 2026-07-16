package crawler

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/DimaMaimesko/web-crawler/internal/fetcher"
)

// fakeFetcher is a deterministic, in-memory Fetcher for tests.
// pages maps a URL to the links it returns; failURLs maps a URL to an error.
type fakeFetcher struct {
	pages    map[string][]string
	failURLs map[string]error
}

func (f fakeFetcher) FetchAndExtract(url string) (fetcher.FetchResult, []string, error) {
	if err, ok := f.failURLs[url]; ok {
		return fetcher.FetchResult{URL: url}, nil, err
	}
	links, ok := f.pages[url]
	if !ok {
		return fetcher.FetchResult{URL: url, StatusCode: 404}, nil, errors.New("not found")
	}
	return fetcher.FetchResult{URL: url, StatusCode: 200}, links, nil
}

// collectURLs returns the sorted set of successfully crawled URLs.
func collectURLs(results []PageResult) []string {
	var urls []string
	for _, r := range results {
		if r.Error == "" {
			urls = append(urls, r.URL)
		}
	}
	sort.Strings(urls)
	return urls
}

func TestCrawl(t *testing.T) {
	tests := []struct {
		name     string
		pages    map[string][]string
		failURLs map[string]error
		seed     string
		maxDepth int
		wantURLs []string
	}{
		{
			name: "single page no links",
			pages: map[string][]string{
				"http://a/": {},
			},
			seed:     "http://a/",
			maxDepth: 2,
			wantURLs: []string{"http://a/"},
		},
		{
			name: "follows links within depth",
			pages: map[string][]string{
				"http://a/":  {"http://a/1", "http://a/2"},
				"http://a/1": {},
				"http://a/2": {},
			},
			seed:     "http://a/",
			maxDepth: 1,
			wantURLs: []string{"http://a/", "http://a/1", "http://a/2"},
		},
		{
			name: "respects max depth",
			pages: map[string][]string{
				"http://a/":  {"http://a/1"},
				"http://a/1": {"http://a/2"},
				"http://a/2": {"http://a/3"},
			},
			seed:     "http://a/",
			maxDepth: 1,
			// depth 0: seed, depth 1: /1, depth 2: /2 exceeds maxDepth -> error (not collected)
			wantURLs: []string{"http://a/", "http://a/1"},
		},
		{
			name: "deduplicates repeated links",
			pages: map[string][]string{
				"http://a/":  {"http://a/1", "http://a/1", "http://a/1"},
				"http://a/1": {"http://a/"}, // cycle back to seed
			},
			seed:     "http://a/",
			maxDepth: 3,
			wantURLs: []string{"http://a/", "http://a/1"},
		},
		{
			name: "records fetch errors but keeps crawling",
			pages: map[string][]string{
				"http://a/":   {"http://a/ok", "http://a/bad"},
				"http://a/ok": {},
			},
			failURLs: map[string]error{
				"http://a/bad": errors.New("boom"),
			},
			seed:     "http://a/",
			maxDepth: 2,
			wantURLs: []string{"http://a/", "http://a/ok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fakeFetcher{pages: tt.pages, failURLs: tt.failURLs}

			// Tiny interval so the rate limiter doesn't slow tests down.
			results := Crawl(f, tt.seed, 4, tt.maxDepth, time.Microsecond)

			got := collectURLs(results)
			if len(got) != len(tt.wantURLs) {
				t.Fatalf("crawled URLs = %v, want %v", got, tt.wantURLs)
			}
			for i := range got {
				if got[i] != tt.wantURLs[i] {
					t.Errorf("crawled URLs = %v, want %v", got, tt.wantURLs)
					break
				}
			}
		})
	}
}

func TestPageResultSuccessClassification(t *testing.T) {
	tests := []struct {
		name        string
		result      PageResult
		wantSuccess bool
	}{
		{"successful page", PageResult{URL: "http://a/1", StatusCode: 200}, true},
		{"fetch error", PageResult{URL: "http://a/2", Error: "connection refused"}, false},
		{"max depth exceeded", PageResult{URL: "http://a/3", Depth: 5, Error: "max depth exceeded"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Error == ""; got != tt.wantSuccess {
				t.Errorf("success = %v, want %v", got, tt.wantSuccess)
			}
		})
	}
}
