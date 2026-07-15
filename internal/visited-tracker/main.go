package visitedTracker

import (
	"fmt"
	"strings"
	"sync"
)

// normalizeURL converts a URL to a canonical form.
func normalizeURL(rawURL string) string {
	u := strings.ToLower(rawURL)
	u = strings.TrimRight(u, "/")
	return u
}

// VisitedTracker tracks which URLs have been visited in a thread-safe way.
type VisitedTracker struct {
	visited map[string]bool
	mu      sync.RWMutex
}

// NewVisitedTracker creates a new empty tracker.
func NewVisitedTracker() *VisitedTracker {
	return &VisitedTracker{
		visited: make(map[string]bool),
	}
}

// HasVisited returns true if the URL has been visited before.
func (v *VisitedTracker) HasVisited(url string) bool {
	normalized := normalizeURL(url)
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.visited[normalized]
}

// MarkVisited marks a URL as visited.
// Returns true if the URL was newly added, false if already visited.
func (v *VisitedTracker) MarkVisited(url string) bool {
	normalized := normalizeURL(url)
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.visited[normalized] {
		return false
	}
	v.visited[normalized] = true
	return true
}

// Count returns the number of unique URLs visited.
func (v *VisitedTracker) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.visited)
}

func main() {
	tracker := NewVisitedTracker()

	// Mark some URLs
	urls := []string{
		"http://gadgetshop/shop/",
		"http://gadgetshop/shop/products",
		"http://gadgetshop/shop/about",
		"http://gadgetshop/shop/",          // duplicate
		"http://gadgetshop/shop/products/", // duplicate (trailing slash)
	}

	for _, u := range urls {
		isNew := tracker.MarkVisited(u)
		if isNew {
			fmt.Printf("NEW:  %s\n", u)
		} else {
			fmt.Printf("SKIP: %s\n", u)
		}
	}

	fmt.Printf("\nTotal unique URLs: %d\n", tracker.Count())

	// Check specific URLs
	fmt.Printf("Has visited /shop/products: %v\n", tracker.HasVisited("http://gadgetshop/shop/products"))
	fmt.Printf("Has visited /shop/missing: %v\n", tracker.HasVisited("http://gadgetshop/shop/missing"))

	// Test concurrent safety
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			url := fmt.Sprintf("http://gadgetshop/shop/page-%d", n%10)
			tracker.MarkVisited(url)
		}(i)
	}
	wg.Wait()

	fmt.Printf("After concurrent adds: %d unique URLs\n", tracker.Count())
	fmt.Println("Done!")
}
