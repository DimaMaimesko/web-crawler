package main

import (
	"fmt"
	"sync"

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

func worker(id int, jobs <-chan CrawlTask, results chan<- PageResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range jobs {
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
		fmt.Println("Worker %d", id)

		results <- pageResult
	}
}

func crawl(seedURL string, maxWorkers int) []PageResult {
	jobs := make(chan CrawlTask, 10)
	results := make(chan PageResult, 10)
	visited := visitedTracker.NewVisitedTracker()

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	visited.MarkVisited(seedURL)
	pending := 1
	jobs <- CrawlTask{URL: seedURL, Depth: 0}

	var allResults []PageResult

	for pending > 0 {
		result := <-results
		pending--
		allResults = append(allResults, result)

		for _, link := range result.Links {
			if visited.MarkVisited(link) {
				pending++
				jobs <- CrawlTask{URL: link, Depth: result.Depth + 1}
			}
		}
	}

	close(jobs)
	return allResults
}

func main() {
	results := crawl(baseURL+"/catalogue/category/books/travel_2/", 3)

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
