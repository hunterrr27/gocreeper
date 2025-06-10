package crawler

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CrawlResult represents a discovered URL with its metadata
type CrawlResult struct {
	URL       string
	Depth     int
	Status    int
	Timestamp time.Time
	Type      string // "html", "css", "js", "image", etc.
}

// Crawler manages the crawling process
type Crawler struct {
	client     *Client
	parser     *Parser
	maxDepth   int
	workers    int
	results    chan CrawlResult
	visited    sync.Map
	queue      chan string
	wg         sync.WaitGroup
	stopChan   chan struct{}
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// CrawlerConfig holds configuration options for the crawler
type CrawlerConfig struct {
	URL               string
	MaxDepth          int
	Workers           int
	Timeout           time.Duration
	MaxRedirects      int
	Headers           map[string]string
	RequestsPerSecond float64
}

// NewCrawler creates a new crawler instance
func NewCrawler(config CrawlerConfig) (*Crawler, error) {
	client, err := NewClient(ClientConfig{
		Timeout:           config.Timeout,
		MaxRedirects:      config.MaxRedirects,
		Headers:           config.Headers,
		BaseURL:           config.URL,
		RequestsPerSecond: config.RequestsPerSecond,
	})
	if err != nil {
		return nil, err
	}

	parser, err := NewParser(config.URL)
	if err != nil {
		return nil, err
	}

	// Create a context with timeout for the entire crawling process
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

	return &Crawler{
		client:     client,
		parser:     parser,
		maxDepth:   config.MaxDepth,
		workers:    config.Workers,
		results:    make(chan CrawlResult, 1000),
		queue:      make(chan string, 1000),
		stopChan:   make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
	}, nil
}

// Start begins the crawling process
func (c *Crawler) Start() <-chan CrawlResult {
	// Start worker pool
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.worker()
	}

	// Start with the initial URL
	c.queue <- c.client.baseURL.String()

	// Start a goroutine to close the results channel when crawling is done
	go func() {
		c.wg.Wait()
		close(c.results)
	}()

	return c.results
}

// Stop gracefully stops the crawler
func (c *Crawler) Stop() {
	c.cancelFunc() // Cancel the context
	close(c.stopChan)
	close(c.queue)
	c.wg.Wait()
}

// worker processes URLs from the queue
func (c *Crawler) worker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.stopChan:
			return
		case urlStr, ok := <-c.queue:
			if !ok {
				return
			}
			c.processURL(urlStr, 0)
		}
	}
}

// processURL handles a single URL
func (c *Crawler) processURL(urlStr string, depth int) {
	// Check if we've already visited this URL
	if _, visited := c.visited.LoadOrStore(urlStr, true); visited {
		return
	}

	// Check depth limit
	if depth > c.maxDepth {
		return
	}

	// Check if URL is in scope
	if !c.client.IsInScope(urlStr) {
		return
	}

	// Fetch the URL using the crawler's context
	resp, err := c.client.Get(c.ctx, urlStr)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Record the result
	c.results <- CrawlResult{
		URL:       urlStr,
		Depth:     depth,
		Status:    resp.StatusCode,
		Timestamp: time.Now(),
		Type:      getContentType(resp.Header.Get("Content-Type")),
	}

	// Process content based on type
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		// Create a new reader for the body since we've already read it
		bodyReader := strings.NewReader(string(body))

		switch getContentType(resp.Header.Get("Content-Type")) {
		case "html":
			urls, err := c.parser.ExtractURLs(bodyReader)
			if err == nil {
				for _, newURL := range urls {
					select {
					case c.queue <- newURL:
						// Process the new URL at depth + 1
						go c.processURL(newURL, depth+1)
					case <-c.ctx.Done():
						return
					default:
						// Queue is full, skip this URL
					}
				}
			}
		case "css":
			urls := c.parser.ExtractCSSURLs(string(body))
			for _, newURL := range urls {
				select {
				case c.queue <- newURL:
					// Process the new URL at depth + 1
					go c.processURL(newURL, depth+1)
				case <-c.ctx.Done():
					return
				default:
					// Queue is full, skip this URL
				}
			}
		case "javascript":
			urls := c.parser.ExtractJSURLs(string(body))
			for _, newURL := range urls {
				select {
				case c.queue <- newURL:
					// Process the new URL at depth + 1
					go c.processURL(newURL, depth+1)
				case <-c.ctx.Done():
					return
				default:
					// Queue is full, skip this URL
				}
			}
		}
	}
}

// getContentType determines the content type from the Content-Type header
func getContentType(contentType string) string {
	switch {
	case strings.Contains(contentType, "text/html"):
		return "html"
	case strings.Contains(contentType, "text/css"):
		return "css"
	case strings.Contains(contentType, "javascript") || strings.Contains(contentType, "application/javascript"):
		return "javascript"
	case strings.Contains(contentType, "image/"):
		return "image"
	default:
		return "other"
	}
}
