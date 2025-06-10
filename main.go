package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hunterrr27/gocreeper/crawler"
)

type outputFormat string

const (
	formatTree outputFormat = "tree"
	formatJSON outputFormat = "json"
	formatCSV  outputFormat = "csv"
)

func main() {
	// Parse command line arguments
	url := flag.String("u", "", "Target URL to crawl (required)")
	depth := flag.Int("d", 3, "Maximum crawl depth")
	workers := flag.Int("c", 10, "Number of concurrent workers")
	timeout := flag.Duration("t", 30*time.Second, "Request timeout")
	rate := flag.Float64("r", 10.0, "Requests per second")
	format := flag.String("o", "tree", "Output format (tree/json/csv)")
	headers := flag.String("H", "", "Custom headers (comma-separated key=value pairs)")
	flag.Parse()

	// Validate required arguments
	if *url == "" {
		fmt.Println("Error: URL is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse custom headers
	headerMap := make(map[string]string)
	if *headers != "" {
		for _, header := range strings.Split(*headers, ",") {
			parts := strings.SplitN(header, "=", 2)
			if len(parts) == 2 {
				headerMap[parts[0]] = parts[1]
			}
		}
	}

	// Set default User-Agent if not provided
	if _, ok := headerMap["User-Agent"]; !ok {
		headerMap["User-Agent"] = "GoCreeper/1.0"
	}

	// Create crawler configuration
	config := crawler.CrawlerConfig{
		URL:              *url,
		MaxDepth:         *depth,
		Workers:          *workers,
		Timeout:          *timeout,
		MaxRedirects:     10,
		Headers:          headerMap,
		RequestsPerSecond: *rate,
	}

	// Initialize crawler
	c, err := crawler.NewCrawler(config)
	if err != nil {
		fmt.Printf("Error initializing crawler: %v\n", err)
		os.Exit(1)
	}

	// Start crawling
	results := c.Start()

	// Handle results based on output format
	switch outputFormat(*format) {
	case formatJSON:
		outputJSON(results)
	case formatCSV:
		outputCSV(results)
	default:
		outputTree(results)
	}

	// Stop the crawler
	c.Stop()
}

func outputJSON(results <-chan crawler.CrawlResult) {
	fmt.Println("[")
	first := true
	for result := range results {
		if !first {
			fmt.Println(",")
		}
		first = false
		json, _ := json.MarshalIndent(result, "", "  ")
		fmt.Print(string(json))
	}
	fmt.Println("\n]")
}

func outputCSV(results <-chan crawler.CrawlResult) {
	fmt.Println("URL,Depth,Status,Timestamp,Type")
	for result := range results {
		fmt.Printf("%s,%d,%d,%s,%s\n",
			result.URL,
			result.Depth,
			result.Status,
			result.Timestamp.Format(time.RFC3339),
			result.Type)
	}
}

func outputTree(results <-chan crawler.CrawlResult) {
	// Group results by depth
	byDepth := make(map[int][]crawler.CrawlResult)
	maxDepth := 0

	for result := range results {
		byDepth[result.Depth] = append(byDepth[result.Depth], result)
		if result.Depth > maxDepth {
			maxDepth = result.Depth
		}
	}

	// Print tree structure
	for depth := 0; depth <= maxDepth; depth++ {
		prefix := strings.Repeat("  ", depth)
		for _, result := range byDepth[depth] {
			status := fmt.Sprintf("[%d]", result.Status)
			fmt.Printf("%s%s %s %s\n", prefix, status, result.Type, result.URL)
		}
	}
} 