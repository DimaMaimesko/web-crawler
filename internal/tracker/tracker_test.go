package tracker

import (
	"fmt"
	"sync"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"lowercase", "HTTP://Example.COM/Path", "http://example.com/path"},
		{"trailing slash", "http://example.com/path/", "http://example.com/path"},
		{"multiple trailing slashes", "http://example.com/path///", "http://example.com/path"},
		{"no change", "http://example.com/path", "http://example.com/path"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeURL(tt.in); got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMarkVisited(t *testing.T) {
	tests := []struct {
		name      string
		urls      []string
		wantNew   []bool // expected return of MarkVisited for each URL in order
		wantCount int
	}{
		{
			name:      "all unique",
			urls:      []string{"http://a/1", "http://a/2", "http://a/3"},
			wantNew:   []bool{true, true, true},
			wantCount: 3,
		},
		{
			name:      "exact duplicate",
			urls:      []string{"http://a/1", "http://a/1"},
			wantNew:   []bool{true, false},
			wantCount: 1,
		},
		{
			name:      "trailing slash duplicate",
			urls:      []string{"http://a/1", "http://a/1/"},
			wantNew:   []bool{true, false},
			wantCount: 1,
		},
		{
			name:      "case-insensitive duplicate",
			urls:      []string{"http://A/Page", "http://a/page"},
			wantNew:   []bool{true, false},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewVisitedTracker()
			for i, u := range tt.urls {
				if got := tr.MarkVisited(u); got != tt.wantNew[i] {
					t.Errorf("MarkVisited(%q) = %v, want %v", u, got, tt.wantNew[i])
				}
			}
			if got := tr.Count(); got != tt.wantCount {
				t.Errorf("Count() = %d, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestHasVisited(t *testing.T) {
	tr := NewVisitedTracker()
	tr.MarkVisited("http://example.com/page")

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"exact match", "http://example.com/page", true},
		{"trailing slash match", "http://example.com/page/", true},
		{"case-insensitive match", "http://Example.com/PAGE", true},
		{"not visited", "http://example.com/missing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tr.HasVisited(tt.url); got != tt.want {
				t.Errorf("HasVisited(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// TestConcurrentMarkVisited reproduces the concurrency check from the old demo.
// Run with -race to detect data races.
func TestConcurrentMarkVisited(t *testing.T) {
	tr := NewVisitedTracker()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			url := fmt.Sprintf("http://gadgetshop/shop/page-%d", n%10)
			tr.MarkVisited(url)
		}(i)
	}
	wg.Wait()

	if got := tr.Count(); got != 10 {
		t.Errorf("Count() after concurrent adds = %d, want 10", got)
	}
}
