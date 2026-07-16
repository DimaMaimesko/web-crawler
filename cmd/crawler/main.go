package main

import (
	"fmt"
	"time"

	"github.com/DimaMaimesko/web-crawler/internal/crawler"
)

const baseURL = "https://books.toscrape.com"

func main() {
	results := crawler.Crawl(crawler.DefaultFetcher(), baseURL+"/", 2, 1, 1*time.Millisecond)

	fmt.Printf("Crawled %d pages:\n", len(results))
	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("  [%d] %s - ERROR: %s\n", r.Depth, r.URL, r.Error)
		} else {
			fmt.Printf("  [%d] %s - %d links\n", r.Depth, r.URL, len(r.Links))
		}
	}

	success := 0
	for _, r := range results {
		if r.Error == "" {
			success++
		}
	}
	fmt.Printf("\nSuccessful: %d, Total: %d\n", success, len(results))
}
