package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// FetchResult holds the result of fetching a URL.
type FetchResult struct {
	URL        string
	Body       string
	StatusCode int
}

// fetchPage makes an HTTP GET request and returns the result.
func fetchPage(rawURL string) (FetchResult, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return FetchResult{URL: rawURL}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchResult{URL: rawURL, StatusCode: resp.StatusCode}, err
	}

	return FetchResult{URL: rawURL, Body: string(body), StatusCode: resp.StatusCode}, nil
}

// extractLinks parses the HTML body and returns all href values from <a> tags.
func extractLinks(body string) ([]string, error) {
	var links []string

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	return links, nil
}

// resolveURL converts a relative link to an absolute URL against base.
func resolveURL(base, link string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}

// FetchAndExtract fetches a URL and returns the result plus resolved links.
// Returns an error if the fetch fails or if the status code is not 200.
func FetchAndExtract(rawURL string) (FetchResult, []string, error) {
	result, err := fetchPage(rawURL)
	if err != nil {
		return result, nil, err
	}

	if result.StatusCode != 200 {
		return result, nil, fmt.Errorf("status %d: %s", result.StatusCode, rawURL)
	}

	rawLinks, _ := extractLinks(result.Body)

	var resolved []string
	for _, link := range rawLinks {
		if link == "" || strings.HasPrefix(link, "#") ||
			strings.HasPrefix(link, "mailto:") ||
			strings.HasPrefix(link, "javascript:") {
			continue
		}
		abs, err := resolveURL(rawURL, link)
		if err != nil {
			continue
		}
		resolved = append(resolved, abs)
	}

	return result, resolved, nil
}
