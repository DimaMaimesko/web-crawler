package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/DimaMaimesko/web-crawler/internal/fetcher"
	"github.com/DimaMaimesko/web-crawler/internal/visited-tracker"
)

const baseURL = "https://books.toscrape.com"

type CrawlTask struct {
	URL   string
	Depth int
}

type PageResult struct {
	URL        string
	StatusCode int
	Links      []string
	Error      string
	Depth      int
}

func workerWithLimits(id int, jobs <-chan CrawlTask, results chan<- PageResult,
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

		result, links, err := fetcher.FetchAndExtract(task.URL)
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

func crawlWithLimits(seedURL string, maxWorkers, maxDepth int, rateInterval time.Duration) []PageResult {
	jobs := make(chan CrawlTask, 100)
	results := make(chan PageResult, 100)
	visited := visitedTracker.NewVisitedTracker()
	limiter := NewRateLimiter(rateInterval)
	defer limiter.Stop()

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go workerWithLimits(i, jobs, results, &wg, maxDepth, limiter)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	visited.MarkVisited(seedURL)
	pending := 1
	// Send the seed in a goroutine so main is free to drain results.
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
				// Send without blocking the drain loop.
				go func(t CrawlTask) {
					jobs <- t
				}(task)
			}
		}
	}

	close(jobs)
	return allResults
}

func main() {
	results := crawlWithLimits(baseURL+"/", 2, 1, 1*time.Millisecond)

	fmt.Printf("Crawled %d pages:\n", len(results))
	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("  [%d] %s - ERROR: %s\n", r.Depth, r.URL, r.Error)
		} else {
			fmt.Printf("  [%d] %s - %d links\n", r.Depth, r.URL, len(r.Links))
		}
	}

	// Count successful pages
	success := 0
	for _, r := range results {
		if r.Error == "" {
			success++
		}
	}
	fmt.Printf("\nSuccessful: %d, Total: %d\n", success, len(results))
}
