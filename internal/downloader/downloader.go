package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sudo-Ivan/website-archiver/config"
	"golang.org/x/net/html"
)

// Download fetches a URL and its dependencies, saving them to the specified output directory.
func Download(ctx context.Context, rawURL string, depth int, outputDir string, cfg *config.Config) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, cfg.DirPerms); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	return downloadRecursive(ctx, parsedURL, parsedURL.Hostname(), depth, outputDir, cfg)
}

func downloadRecursive(ctx context.Context, currentURL *url.URL, baseDomain string, depth int, outputDir string, cfg *config.Config) error {
	if depth < 0 {
		return nil
	}

	if currentURL.Hostname() != baseDomain && baseDomain != "" {
		// Do not download external domains recursively
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", currentURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", currentURL.String(), err)
	}

	client := &http.Client{Timeout: cfg.HTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", currentURL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch %s: status code %d", currentURL.String(), resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html")

	filePath := filepath.Join(outputDir, getPathFromURL(currentURL, isHTML))
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, cfg.DirPerms); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
	}

	file, err := os.Create(filePath) // #nosec G304 - filePath is constructed from a sanitized URL path
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	if isHTML {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body for %s: %w", currentURL.String(), err)
		}
		// Write the original content to the file
		if _, err := file.Write(bodyBytes); err != nil {
			return fmt.Errorf("failed to write content to %s: %w", filePath, err)
		}

		// Parse the HTML for links
		doc, err := html.Parse(strings.NewReader(string(bodyBytes)))
		if err != nil {
			return fmt.Errorf("failed to parse HTML for %s: %w", currentURL.String(), err)
		}

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				for i, a := range n.Attr {
					var link string
					switch a.Key {
					case "href", "src":
						link = a.Val
					case "poster": // For video poster images
						link = a.Val
					default:
						continue
					}

					if link == "" || strings.HasPrefix(link, "#") || strings.HasPrefix(link, "mailto:") || strings.HasPrefix(link, "tel:") {
						continue
					}

					resolvedURL := resolveURL(currentURL, link)
					if resolvedURL != nil && resolvedURL.String() != currentURL.String() {
						go func(u *url.URL) {
							if err := downloadRecursive(ctx, u, baseDomain, depth-1, outputDir, cfg); err != nil {
								// Log error, but don't stop the main download
							}
						}(resolvedURL)

						// Convert links in the HTML to relative paths or updated paths
						newLink := getPathFromURL(resolvedURL, strings.Contains(link, ".html") || strings.Contains(link, ".htm"))
						n.Attr[i].Val = newLink
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)

		// Re-write the HTML with updated links
		var buf strings.Builder
		if err := html.Render(&buf, doc); err != nil {
			return fmt.Errorf("failed to render HTML with updated links for %s: %w", filePath, err)
		}
		if err := os.WriteFile(filePath, []byte(buf.String()), cfg.FilePerms); err != nil {
			return fmt.Errorf("failed to write updated HTML to %s: %w", filePath, err)
		}

	} else {
		if _, err := io.Copy(file, resp.Body); err != nil {
			return fmt.Errorf("failed to save %s to %s: %w", currentURL.String(), filePath, err)
		}
	}

	return nil
}

func getPathFromURL(u *url.URL, isHTML bool) string {
	path := u.Path
	if strings.HasSuffix(path, "/") || path == "" {
		if isHTML {
			path += "index.html"
		} else {
			path += "index" // Default for non-HTML if it's a directory
		}
	}
	// Ensure the path is relative and clean to prevent directory traversal
	cleanPath := filepath.Clean(strings.TrimPrefix(path, "/"))
	if strings.HasPrefix(cleanPath, "..") {
		return "" // Or handle as an error, depending on desired behavior
	}
	return cleanPath
}

func resolveURL(baseURL *url.URL, ref string) *url.URL {
	refURL, err := url.Parse(ref)
	if err != nil {
		return nil
	}
	return baseURL.ResolveReference(refURL)
}
