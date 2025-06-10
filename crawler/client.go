package crawler

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

// Client represents our custom HTTP client with rate limiting and configurable options
type Client struct {
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	headers     map[string]string
	baseURL     *url.URL
}

// ClientConfig holds configuration options for the HTTP client
type ClientConfig struct {
	Timeout           time.Duration
	MaxRedirects      int
	Headers           map[string]string
	BaseURL           string
	RequestsPerSecond float64
}

// NewClient creates a new custom HTTP client with the given configuration
func NewClient(config ClientConfig) (*Client, error) {
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}

	// Create custom transport with timeouts
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Create HTTP client with custom transport
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Create rate limiter with a minimum of 1 request per second
	rps := config.RequestsPerSecond
	if rps < 1.0 {
		rps = 1.0
	}
	// Create a limiter with a burst of 1 to ensure strict rate limiting
	limiter := rate.NewLimiter(rate.Limit(rps), 1)

	return &Client{
		httpClient:  httpClient,
		rateLimiter: limiter,
		headers:     config.Headers,
		baseURL:     baseURL,
	}, nil
}

// Get performs a GET request with rate limiting and custom headers
func (c *Client) Get(ctx context.Context, urlStr string) (*http.Response, error) {
	// Wait for rate limiter with the provided context
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Parse URL
	reqURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Perform request
	return c.httpClient.Do(req)
}

// IsInScope checks if a URL is within the target domain scope
func (c *Client) IsInScope(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return parsedURL.Hostname() == c.baseURL.Hostname()
}
