package fetcher

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "no links",
			body: `<html><body><p>hello</p></body></html>`,
			want: nil,
		},
		{
			name: "single link",
			body: `<a href="/about">About</a>`,
			want: []string{"/about"},
		},
		{
			name: "multiple links",
			body: `<a href="/a">A</a><a href="/b">B</a><a href="http://x.com">X</a>`,
			want: []string{"/a", "/b", "http://x.com"},
		},
		{
			name: "nested links",
			body: `<div><ul><li><a href="/deep">Deep</a></li></ul></div>`,
			want: []string{"/deep"},
		},
		{
			name: "anchor without href is ignored",
			body: `<a name="top">Top</a><a href="/real">Real</a>`,
			want: []string{"/real"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractLinks(tt.body)
			if err != nil {
				t.Fatalf("extractLinks returned error: %v", err)
			}
			if !equalStrings(got, tt.want) {
				t.Errorf("extractLinks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		link    string
		want    string
		wantErr bool
	}{
		{
			name: "absolute path against host",
			base: "http://example.com/catalogue/",
			link: "/about",
			want: "http://example.com/about",
		},
		{
			name: "relative path",
			base: "http://example.com/catalogue/page.html",
			link: "next.html",
			want: "http://example.com/catalogue/next.html",
		},
		{
			name: "dot-slash relative",
			base: "http://example.com/catalogue/",
			link: "./book_1/index.html",
			want: "http://example.com/catalogue/book_1/index.html",
		},
		{
			name: "parent relative",
			base: "http://example.com/catalogue/category/",
			link: "../../index.html",
			want: "http://example.com/index.html",
		},
		{
			name: "already absolute",
			base: "http://example.com/",
			link: "http://other.com/page",
			want: "http://other.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveURL(tt.base, tt.link)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.base, tt.link, got, tt.want)
			}
		})
	}
}

func TestFetchAndExtract(t *testing.T) {
	// A local server so the test is fully offline.
	mux := http.NewServeMux()
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<html><body>
				<a href="/next">Next</a>
				<a href="sub/child.html">Child</a>
				<a href="#section">Anchor</a>
				<a href="mailto:me@example.com">Mail</a>
				<a href="javascript:void(0)">JS</a>
				<a href="">Empty</a>
			</body></html>
		`))
	})
	mux.HandleFunc("/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Run("resolves and filters links", func(t *testing.T) {
		result, links, err := FetchAndExtract(srv.URL + "/page")
		if err != nil {
			t.Fatalf("FetchAndExtract returned error: %v", err)
		}
		if result.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want 200", result.StatusCode)
		}

		want := []string{
			srv.URL + "/next",
			srv.URL + "/sub/child.html",
		}
		got := append([]string(nil), links...)
		sort.Strings(got)
		sort.Strings(want)
		if !equalStrings(got, want) {
			t.Errorf("links = %v, want %v (anchor/mailto/javascript/empty must be filtered)", got, want)
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		_, _, err := FetchAndExtract(srv.URL + "/missing")
		if err == nil {
			t.Error("expected error for 404 response, got nil")
		}
	})

	t.Run("unreachable host returns error", func(t *testing.T) {
		// Closed server address -> connection error.
		_, _, err := FetchAndExtract("http://127.0.0.1:0/nope")
		if err == nil {
			t.Error("expected error for unreachable host, got nil")
		}
	})
}

// equalStrings compares two string slices for equality, treating nil and
// empty as equal.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
