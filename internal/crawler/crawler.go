package crawler

import (
	"sync"
	"time"

	"github.com/DimaMaimesko/web-crawler/internal/fetcher"
	"github.com/DimaMaimesko/web-crawler/internal/tracker"
)

// Fetcher retrieves a page and returns its result plus the links found on it.
// It is defined here (by the consumer) so the crawler can be tested with a
// fake implementation, decoupled from the real network-backed fetcher.
type Fetcher interface {
	FetchAndExtract(url string) (fetcher.FetchResult, []string, error)
}

// realFetcher is the production adapter that delegates to the fetcher package.
type realFetcher struct{}

func (realFetcher) FetchAndExtract(url string) (fetcher.FetchResult, []string, error) {
	return fetcher.FetchAndExtract(url)
}

// DefaultFetcher returns the production Fetcher backed by real HTTP requests.
func DefaultFetcher() Fetcher {
	return realFetcher{}
}

// CrawlTask is a single unit of work: a URL to fetch at a given depth.
type CrawlTask struct {
	URL   string
	Depth int
}

// PageResult is the outcome of crawling a single page.
type PageResult struct {
	URL        string
	StatusCode int
	Links      []string
	Error      string
	Depth      int
}

func worker(f Fetcher, jobs <-chan CrawlTask, results chan<- PageResult,
	wg *sync.WaitGroup, maxDepth int, limiter *RateLimiter) {
	defer wg.Done()

	for task := range jobs {
		if task.Depth > maxDepth {
			results <- PageResult{
				URL:   task.URL,
				Depth: task.Depth,
				Error: "max depth exceeded",
			}
			continue
		}

		limiter.Wait()

		result, links, err := f.FetchAndExtract(task.URL)
		pageResult := PageResult{
			URL:        task.URL,
			StatusCode: result.StatusCode,
			Links:      links,
			Depth:      task.Depth,
		}
		if err != nil {
			pageResult.Error = err.Error()
		}

		results <- pageResult
	}
}

// Crawl starts from seedURL and crawls concurrently using the given Fetcher,
// respecting the worker count, max depth, and rate interval.
func Crawl(f Fetcher, seedURL string, maxWorkers, maxDepth int, rateInterval time.Duration) []PageResult {
	jobs := make(chan CrawlTask, 100)
	results := make(chan PageResult, 100)
	visited := tracker.NewVisitedTracker()
	limiter := NewRateLimiter(rateInterval)
	defer limiter.Stop()

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(f, jobs, results, &wg, maxDepth, limiter)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	visited.MarkVisited(seedURL)
	pending := 1
	go func() { jobs <- CrawlTask{URL: seedURL, Depth: 0} }()

	var allResults []PageResult

	for pending > 0 {
		result := <-results
		pending--

		if result.Error == "" {
			allResults = append(allResults, result)
		}

		for _, link := range result.Links {
			if visited.MarkVisited(link) {
				pending++
				task := CrawlTask{URL: link, Depth: result.Depth + 1}
				go func(t CrawlTask) {
					jobs <- t
				}(task)
			}
		}
	}

	close(jobs)
	return allResults
}
