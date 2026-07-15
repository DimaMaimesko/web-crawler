package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const baseURL = "https://books.toscrape.com"

// FetchResult holds the result of fetching a URL
type FetchResult struct {
	URL        string
	Body       string
	StatusCode int
}

// fetchPage makes an HTTP GET request and returns the result.
func fetchPage(url string) (FetchResult, error) {
	resp, err := http.Get(url)
	if err != nil {
		return FetchResult{URL: url}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchResult{URL: url, StatusCode: resp.StatusCode}, err
	}

	return FetchResult{URL: url, Body: string(body), StatusCode: resp.StatusCode}, nil
}

// extractLinks finds all href="..." values in an HTML body.
func extractLinks(body string) []string {
	var links []string
	search := `href="`
	remaining := body

	for {
		idx := strings.Index(remaining, search)
		if idx == -1 {
			break
		}
		start := idx + len(search)
		remaining = remaining[start:]

		end := strings.Index(remaining, `"`)
		if end == -1 {
			break
		}

		link := remaining[:end]
		links = append(links, link)
		remaining = remaining[end:]
	}

	return links
}

// resolveURL converts a relative link to an absolute URL using the base URL.
func resolveURL(base, link string) string {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}
	base = strings.TrimRight(base, "/")
	if strings.HasPrefix(link, "/") {
		parts := strings.SplitN(base, "/", 4)
		if len(parts) >= 3 {
			return parts[0] + "//" + parts[2] + link
		}
	}
	return base + "/" + link
}

// FetchAndExtract fetches a URL and returns the result plus resolved links.
// Returns an error if the fetch fails or if the status code is not 200.
func FetchAndExtract(url string) (FetchResult, []string, error) {
	result, err := fetchPage(url)
	if err != nil {
		return result, nil, err
	}

	if result.StatusCode != 200 {
		return result, nil, fmt.Errorf("status %d: %s", result.StatusCode, url)
	}

	rawLinks := extractLinks(result.Body)

	var resolved []string
	for _, link := range rawLinks {
		resolved = append(resolved, resolveURL(url, link))
	}

	return result, resolved, nil
}

func main() {
	// Test fetchPage
	result, err := fetchPage(baseURL + "/catalogue/category/books/travel_2/")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Fetched %s (status %d, %d bytes)\n", result.URL, result.StatusCode, len(result.Body))
	}

	// Test extractLinks
	links := extractLinks(result.Body)
	fmt.Printf("Extracted links: %v\n", links)

	// Test 404
	result, err = fetchPage(baseURL + "/catalogue/category/books/travel_3/")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Missing page: status=%d\n", result.StatusCode)
	}

	// Test resolveURL
	fmt.Printf("Resolve relative: %s\n", resolveURL(baseURL, "/catalogue/category/books/travel_2"))
	fmt.Printf("Resolve absolute: %s\n", resolveURL(baseURL, "http://other.com/page"))

	// Test fetchAndExtract
	result, resolved, err := FetchAndExtract(baseURL + "/catalogue/category/books/travel_2/")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Home page has %d links:\n", len(resolved))
		for _, link := range resolved {
			fmt.Printf("  -> %s\n", link)
		}
	}

	// Test fetchAndExtract on about page
	result, resolved, err = FetchAndExtract(baseURL + "/catalogue/category/books/travel_2/")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("About page has %d links\n", len(resolved))
	}

	// Test fetchAndExtract on non-existent page
	_, _, err = FetchAndExtract(baseURL + "/shop/nonexistent")
	if err != nil {
		fmt.Printf("Fetch error: %v\n", err)
	}
}
