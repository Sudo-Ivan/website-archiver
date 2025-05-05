// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

// Package main implements a web page downloader that can fetch pages directly or from the Wayback Machine,
// with support for creating ZIM files for offline viewing.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Sudo-Ivan/website-archiver/config"
	"github.com/Sudo-Ivan/website-archiver/pkg"
)

//go:embed default.png
var embeddedDefaultPNG []byte

// CDXResponse represents a snapshot from the Wayback Machine's CDX API.
type CDXResponse struct {
	Timestamp string `json:"timestamp"`
	Original  string `json:"original"`
	Mimetype  string `json:"mimetype"`
	Status    string `json:"status"`
	Digest    string `json:"digest"`
	Length    string `json:"length"`
}

// DownloadResult represents the result of a download attempt.
type DownloadResult struct {
	URL       string
	Error     error
	OutputDir string
	Timestamp string
}

// Snapshot represents a downloaded snapshot.
type Snapshot struct {
	Timestamp string
	URL       string
	Path      string
}

// validateURL checks if a URL is valid and uses either HTTP or HTTPS scheme.
func validateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}
	return nil
}

// downloadWithWget downloads a URL using wget with specified depth and output directory.
func downloadWithWget(ctx context.Context, url string, depth int, outputDir string, cfg *config.Config) error {
	if depth < pkg.ZeroDepth || depth > cfg.MaxDepth {
		return fmt.Errorf("depth must be between %d and %d", pkg.ZeroDepth, cfg.MaxDepth)
	}

	args := []string{
		"--no-clobber",
		"--html-extension",
		"--convert-links",
		"--restrict-file-names=windows",
		"--domains", getDomain(url),
		"--no-parent",
		"--directory-prefix=" + outputDir,
	}

	if depth == pkg.ZeroDepth {
		args = append(args, "--page-requisites")
	} else {
		args = append(args, "--recursive")
		args = append(args, "--level="+fmt.Sprintf("%d", depth))
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "wget", args...) // #nosec G204 - wget args are validated
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// getDomain extracts the domain name from a URL.
func getDomain(url string) string {
	domain := strings.TrimPrefix(url, "http://")
	domain = strings.TrimPrefix(domain, "https://")

	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	return domain
}

// parseCDXResponse parses the raw CDX API response into a slice of CDXResponse
func parseCDXResponse(rawResponse [][]string) ([]CDXResponse, error) {
	if len(rawResponse) < pkg.MinCDXRows {
		return nil, fmt.Errorf("no snapshots found")
	}

	snapshots := make([]CDXResponse, pkg.ZeroLength, len(rawResponse)-pkg.OneLength)
	for _, row := range rawResponse[pkg.OneLength:] {
		if len(row) >= pkg.MinCDXFields {
			snapshots = append(snapshots, CDXResponse{
				Timestamp: row[pkg.CDXTimestampIndex],
				Original:  row[pkg.CDXOriginalIndex],
				Mimetype:  row[pkg.CDXMimetypeIndex],
				Status:    row[pkg.CDXStatusIndex],
				Digest:    row[pkg.CDXDigestIndex],
				Length:    row[pkg.CDXLengthIndex],
			})
		}
	}
	return snapshots, nil
}

// getCDXSnapshots retrieves snapshots for a given URL from the Wayback Machine's CDX API.
func getCDXSnapshots(ctx context.Context, url string, cfg *config.Config) ([]CDXResponse, error) {
	cdxURL := fmt.Sprintf("%s?url=%s&output=json&fl=timestamp,original,mimetype,status,digest,length", cfg.WaybackAPIURL, url)

	req, err := http.NewRequestWithContext(ctx, "GET", cdxURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDX request: %w", err)
	}

	client := &http.Client{Timeout: cfg.HTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CDX data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CDX response: %w", err)
	}

	var rawResponse [][]string
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse CDX response: %w", err)
	}

	return parseCDXResponse(rawResponse)
}

// tryConvertImage attempts to convert and resize an image to PNG format
func tryConvertImage(srcPath, domainDir string) (string, error) {
	pngPath := filepath.Join(domainDir, pkg.IllustrationPNG)
	cmd := exec.Command(pkg.ConvertCmd, srcPath, pkg.ResizeFlag, pkg.ResizeSize, pngPath) // #nosec G204 - convert args are validated
	if err := cmd.Run(); err != nil {
		return pkg.EmptyString, err
	}
	return filepath.Rel(domainDir, pngPath)
}

// findImageInPatterns searches for images matching the given patterns in the domain directory
func findImageInPatterns(domainDir string, patterns []string) (string, error) {
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(domainDir, pattern))
		if err != nil || len(matches) == pkg.ZeroLength {
			continue
		}
		return matches[pkg.FirstIndex], nil
	}
	return pkg.EmptyString, fmt.Errorf("no images found matching patterns")
}

// convertDefaultImage converts the default image to the required format
func convertDefaultImage(domainDir string) (string, error) {
	defaultDst := filepath.Join(domainDir, pkg.IllustrationPNG)
	if _, err := os.Stat(pkg.DefaultPNG); err == nil {
		cmd := exec.Command(pkg.ConvertCmd, pkg.DefaultPNG, pkg.ResizeFlag, pkg.ResizeSize, defaultDst) // #nosec G204 - convert args are validated
		if err := cmd.Run(); err != nil {
			return pkg.EmptyString, fmt.Errorf("failed to convert %s: %w", pkg.DefaultPNG, err)
		}
		return filepath.Rel(domainDir, defaultDst)
	}
	// If not found on disk, use embedded
	if len(embeddedDefaultPNG) > 0 {
		if err := os.WriteFile(defaultDst, embeddedDefaultPNG, pkg.FilePerms); err != nil {
			return pkg.EmptyString, fmt.Errorf("failed to write embedded default.png: %w", err)
		}
		cmd := exec.Command(pkg.ConvertCmd, defaultDst, pkg.ResizeFlag, pkg.ResizeSize, defaultDst) // #nosec G204 - convert args are validated
		if err := cmd.Run(); err != nil {
			return pkg.EmptyString, fmt.Errorf("failed to convert embedded default.png: %w", err)
		}
		return filepath.Rel(domainDir, defaultDst)
	}
	return pkg.EmptyString, fmt.Errorf("no suitable illustration found and %s is not available", pkg.DefaultPNG)
}

// findOrCreateIllustration attempts to find an illustration (image) for a given domain,
// or creates one from a default image if none is found.
func findOrCreateIllustration(outputDir, domain string) (string, error) {
	domainDir := filepath.Join(outputDir, domain)

	imagePatterns := []string{
		"*.png", "*.jpg", "*.jpeg", "*.ico", "*.gif",
		"favicon.ico", "favicon.png", "logo.png", "logo.jpg",
	}

	srcPath, err := findImageInPatterns(domainDir, imagePatterns)
	if err == nil {
		return tryConvertImage(srcPath, domainDir)
	}

	return convertDefaultImage(domainDir)
}

// createSnapshotSelectionPage generates an HTML page that allows the user to select from available snapshots.
func createSnapshotSelectionPage(snapshots []Snapshot, outputDir string) error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Available Snapshots</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        .snapshot {
            border: 1px solid #ddd;
            margin: 10px 0;
            padding: 15px;
            border-radius: 5px;
        }
        .snapshot:hover {
            background-color: #f5f5f5;
        }
        .timestamp {
            color: #666;
            font-size: 0.9em;
        }
        a {
            color: #0066cc;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #eee;
            padding-bottom: 10px;
        }
    </style>
</head>
<body>
    <h1>Available Snapshots</h1>
    <div class="snapshots">
`

	for _, snapshot := range snapshots {
		html += fmt.Sprintf(`
        <div class="snapshot">
            <a href="%s/index.html">
                <strong>Snapshot from %s</strong>
                <div class="timestamp">%s</div>
            </a>
        </div>`, snapshot.Path, snapshot.Timestamp, snapshot.Timestamp)
	}

	html += `
    </div>
</body>
</html>`

	return os.WriteFile(filepath.Join(outputDir, pkg.IndexHTML), []byte(html), pkg.FilePerms) // #nosec G306 - file needs to be readable by web server
}

// downloadSnapshot downloads a specific snapshot from the Wayback Machine
func downloadSnapshot(ctx context.Context, snapshot string, url string, depth int, outputDir string, cfg *config.Config) error {
	waybackURL := fmt.Sprintf(pkg.WaybackURLFormat, snapshot, url)
	slog.Info("Downloading specific snapshot", pkg.LogTimestamp, snapshot, pkg.LogURL, url)
	return downloadWithWget(ctx, waybackURL, depth, outputDir, cfg)
}

// downloadAllSnapshots downloads all available snapshots for a URL
func downloadAllSnapshots(ctx context.Context, snapshots []CDXResponse, url string, depth int, outputDir string, cfg *config.Config) []Snapshot {
	var downloadedSnapshots []Snapshot
	for _, snapshot := range snapshots {
		snapshotDir := filepath.Join(outputDir, snapshot.Timestamp)
		if err := os.MkdirAll(snapshotDir, cfg.DirPerms); err != nil {
			slog.Warn("Failed to create directory for snapshot", pkg.LogError, err, pkg.LogTimestamp, snapshot.Timestamp)
			continue
		}

		waybackURL := fmt.Sprintf(pkg.WaybackURLFormat, snapshot.Timestamp, url)
		if err := downloadWithWget(ctx, waybackURL, depth, snapshotDir, cfg); err != nil {
			slog.Warn("Failed to download snapshot", pkg.LogError, err, pkg.LogTimestamp, snapshot.Timestamp)
			continue
		}

		downloadedSnapshots = append(downloadedSnapshots, Snapshot{
			Timestamp: snapshot.Timestamp,
			URL:       waybackURL,
			Path:      snapshot.Timestamp,
		})
	}
	return downloadedSnapshots
}

// createZIMFile creates a ZIM file from the downloaded content
func createZIMFile(ctx context.Context, outputDir, url string, downloadedSnapshots []Snapshot) error {
	currentDate := time.Now().Format("20060102")
	zimFile := filepath.Join(filepath.Dir(outputDir), fmt.Sprintf("%s_%s.zim", getDomain(url), currentDate))
	slog.Info("Creating ZIM file", "file", zimFile)

	illustration, err := findOrCreateIllustration(outputDir, getDomain(url))
	if err != nil {
		return fmt.Errorf("failed to find or create illustration: %w", err)
	}

	welcomePage := pkg.IndexHTML
	if len(downloadedSnapshots) == pkg.OneLength {
		welcomePage = filepath.Join(getDomain(url), pkg.IndexHTML)
	}

	cmd := exec.CommandContext(ctx, "zimwriterfs", // #nosec G204 - zimwriterfs args are validated
		"--welcome", welcomePage,
		"--illustration", filepath.Join(getDomain(url), illustration),
		"--language", "eng",
		"--title", getDomain(url),
		"--name", getDomain(url),
		"--description", fmt.Sprintf("Archive of %s%s", url, func() string {
			if len(downloadedSnapshots) > pkg.OneLength {
				return fmt.Sprintf(" with %d snapshots", len(downloadedSnapshots))
			}
			return pkg.EmptyString
		}()),
		"--longDescription", fmt.Sprintf("Offline archive of %s created with website-archiver%s", url, func() string {
			if len(downloadedSnapshots) > pkg.OneLength {
				return fmt.Sprintf(". Contains %d snapshots.", len(downloadedSnapshots))
			}
			return pkg.EmptyString
		}()),
		"--creator", "website-archiver",
		"--publisher", "website-archiver",
		"--withoutFTIndex",
		".",
		zimFile,
	)
	cmd.Dir = outputDir // Set the working directory to outputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create ZIM file: %w", err)
	}
	return nil
}

// downloadCurrentVersion attempts to download the current version of a URL
func downloadCurrentVersion(ctx context.Context, url string, depth int, outputDir string, cfg *config.Config) ([]Snapshot, error) {
	if err := downloadWithWget(ctx, url, depth, outputDir, cfg); err != nil {
		return nil, err
	}
	return []Snapshot{{
		Timestamp: "Current",
		URL:       url,
		Path:      getDomain(url),
	}}, nil
}

// downloadArchivedVersion downloads an archived version of a URL
func downloadArchivedVersion(ctx context.Context, url string, depth int, outputDir string, allSnapshots bool, cfg *config.Config) ([]Snapshot, error) {
	snapshots, err := getCDXSnapshots(ctx, url, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	if len(snapshots) == pkg.ZeroLength {
		return nil, fmt.Errorf("no archived versions available")
	}

	if allSnapshots {
		slog.Info("Found archived versions", "count", len(snapshots), pkg.LogURL, url)
		return downloadAllSnapshots(ctx, snapshots, url, depth, outputDir, cfg), nil
	}

	waybackURL := fmt.Sprintf(pkg.WaybackURLFormat, snapshots[pkg.FirstIndex].Timestamp, url)
	slog.Info("Downloading most recent archived version", pkg.LogTimestamp, snapshots[pkg.FirstIndex].Timestamp, pkg.LogURL, url)

	if err := downloadWithWget(ctx, waybackURL, depth, outputDir, cfg); err != nil {
		return nil, fmt.Errorf("failed to download archived version: %w", err)
	}

	return []Snapshot{{
		Timestamp: snapshots[pkg.FirstIndex].Timestamp,
		URL:       waybackURL,
		Path:      getDomain(url),
	}}, nil
}

// handleSpecificSnapshot handles downloading a specific snapshot
func handleSpecificSnapshot(ctx context.Context, specificSnapshot, url string, depth int, outputDir string, cfg *config.Config) ([]Snapshot, error) {
	if err := downloadSnapshot(ctx, specificSnapshot, url, depth, outputDir, cfg); err != nil {
		slog.Error("Failed to download snapshot", pkg.LogError, err, pkg.LogURL, url)
		return nil, fmt.Errorf("failed to download snapshot: %w", err)
	}

	return []Snapshot{{
		Timestamp: specificSnapshot,
		URL:       fmt.Sprintf(pkg.WaybackURLFormat, specificSnapshot, url),
		Path:      getDomain(url),
	}}, nil
}

// handleCurrentOrArchivedVersion attempts to download current version first, then falls back to archived version
func handleCurrentOrArchivedVersion(ctx context.Context, url string, depth int, outputDir string, allSnapshots bool, cfg *config.Config) ([]Snapshot, error) {
	slog.Info("Attempting direct download", pkg.LogURL, url)
	downloadedSnapshots, err := downloadCurrentVersion(ctx, url, depth, outputDir, cfg)
	if err != nil {
		slog.Warn("Direct download failed, attempting archived versions", pkg.LogError, err, pkg.LogURL, url)
		downloadedSnapshots, err = downloadArchivedVersion(ctx, url, depth, outputDir, allSnapshots, cfg)
		if err != nil {
			slog.Error("Failed to download archived version", pkg.LogError, err, pkg.LogURL, url)
			return nil, err
		}
	}
	return downloadedSnapshots, nil
}

// handleDownloadResult handles the result of a download attempt
func handleDownloadResult(url, outputDir string, err error, results chan<- DownloadResult) {
	if err != nil {
		results <- DownloadResult{URL: url, Error: err}
		if removeErr := os.RemoveAll(outputDir); removeErr != nil {
			slog.Warn("Failed to remove directory after error", pkg.LogError, removeErr, "dir", outputDir)
		}
		return
	}
	results <- DownloadResult{URL: url, OutputDir: outputDir}
}

// handlePostDownloadTasks handles tasks after successful download
func handlePostDownloadTasks(ctx context.Context, downloadedSnapshots []Snapshot, outputDir, url string, createZim bool) {
	if len(downloadedSnapshots) > pkg.OneLength {
		if err := createSnapshotSelectionPage(downloadedSnapshots, outputDir); err != nil {
			slog.Warn("Failed to create selection page", pkg.LogError, err)
		}
	}

	if createZim {
		if err := createZIMFile(ctx, outputDir, url, downloadedSnapshots); err != nil {
			slog.Warn("Failed to create ZIM file", pkg.LogError, err)
		}
	}

	if err := os.RemoveAll(outputDir); err != nil {
		slog.Warn("Failed to remove directory", pkg.LogError, err, "dir", outputDir)
	}
}

// processURL downloads a URL, either directly or from the Wayback Machine, and optionally creates a ZIM file.
func processURL(ctx context.Context, url string, depth int, createZim bool, allSnapshots bool, specificSnapshot string, results chan<- DownloadResult, cfg *config.Config) {
	timestampStr := time.Now().Format("20060102_150405")
	outputDir := filepath.Join(cfg.OutputDir, getDomain(url)+"_"+timestampStr)

	if err := os.MkdirAll(outputDir, cfg.DirPerms); err != nil {
		slog.Error("Failed to create output directory", pkg.LogError, err, pkg.LogURL, url)
		results <- DownloadResult{URL: url, Error: fmt.Errorf("failed to create output directory: %w", err)}
		return
	}

	var downloadedSnapshots []Snapshot
	var err error

	if specificSnapshot != pkg.EmptyString {
		downloadedSnapshots, err = handleSpecificSnapshot(ctx, specificSnapshot, url, depth, outputDir, cfg)
	} else {
		downloadedSnapshots, err = handleCurrentOrArchivedVersion(ctx, url, depth, outputDir, allSnapshots, cfg)
	}

	if err != nil {
		handleDownloadResult(url, outputDir, err, results)
		return
	}

	handlePostDownloadTasks(ctx, downloadedSnapshots, outputDir, url, createZim)
	handleDownloadResult(url, outputDir, nil, results)
}

// validateAndParseArgs validates URLs and parses command line arguments
func validateAndParseArgs() (urls []string, depth int, createZim bool, allSnapshots bool, specificSnapshot string, err error) {
	flag.BoolVar(&createZim, "zim", false, "Create ZIM file from downloaded content")
	flag.BoolVar(&createZim, "z", false, "Create ZIM file from downloaded content (shorthand)")
	flag.BoolVar(&allSnapshots, "all-snapshots", false, "Download all available snapshots")
	flag.BoolVar(&allSnapshots, "as", false, "Download all available snapshots (shorthand)")
	flag.StringVar(&specificSnapshot, "snapshot", pkg.EmptyString, "Download a specific snapshot (format: YYYYMMDDHHMMSS)")
	flag.StringVar(&specificSnapshot, "s", pkg.EmptyString, "Download a specific snapshot (format: YYYYMMDDHHMMSS) (shorthand)")
	flag.Parse()

	args := flag.Args()
	if len(args) < pkg.OneLength {
		return nil, pkg.ZeroDepth, false, false, pkg.EmptyString, fmt.Errorf("no URLs provided")
	}

	depth = pkg.ZeroDepth
	lastArg := args[len(args)-pkg.OneLength]

	if depthVal, err := fmt.Sscanf(lastArg, "%d", &depth); err == nil && depthVal == pkg.OneLength {
		urls = args[:len(args)-pkg.OneLength]
	} else {
		urls = args
	}

	for _, url := range urls {
		if err := validateURL(url); err != nil {
			return nil, pkg.ZeroDepth, false, false, pkg.EmptyString, fmt.Errorf("invalid URL %s: %w", url, err)
		}
	}

	return urls, depth, createZim, allSnapshots, specificSnapshot, nil
}

// processResults processes download results and prints a summary
func processResults(results <-chan DownloadResult, totalURLs int) {
	successCount := pkg.ZeroCount
	for result := range results {
		if result.Error != nil {
			slog.Error("Failed to download", pkg.LogError, result.Error, pkg.LogURL, result.URL)
		} else {
			slog.Info("Successfully downloaded", pkg.LogURL, result.URL, "outputDir", result.OutputDir)
			successCount++
		}
	}

	slog.Info("Download Summary",
		"totalURLs", totalURLs,
		"successful", successCount,
		"failed", totalURLs-successCount,
	)
}

// main is the entry point of the program. It parses command-line arguments,
// validates URLs, and initiates the download process.
func main() {
	// Initialize configuration
	cfg := config.New()

	urls, depth, createZim, allSnapshots, specificSnapshot, err := validateAndParseArgs()
	if err != nil {
		slog.Error("Failed to parse arguments", pkg.LogError, err)
		fmt.Println("Usage: website-archiver [--zim|-z] [--all-snapshots|-as] [--snapshot|-s YYYYMMDDHHMMSS] <url1> [url2] [url3] ... [depth]")
		fmt.Println("Example: website-archiver --zim --all-snapshots https://example.com")
		fmt.Println("Example: website-archiver --zim --snapshot 20230101000000 https://example.com")
		os.Exit(pkg.ExitFailure)
	}

	if createZim {
		if _, err := exec.LookPath("zimwriterfs"); err != nil {
			slog.Error("zimwriterfs not found in PATH", pkg.LogError, err)
			os.Exit(pkg.ExitFailure)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPTimeout*time.Duration(len(urls)))
	defer cancel()

	results := make(chan DownloadResult, len(urls))
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			processURL(ctx, url, depth, createZim, allSnapshots, specificSnapshot, results, cfg)
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	processResults(results, len(urls))
	os.Exit(pkg.ExitSuccess)
}
