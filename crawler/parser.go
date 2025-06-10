package crawler

import (
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Parser handles the extraction of URLs from various content types
type Parser struct {
	baseURL *url.URL
}

// NewParser creates a new parser instance
func NewParser(baseURL string) (*Parser, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Parser{baseURL: parsedURL}, nil
}

// ExtractURLs parses HTML content and extracts all relevant URLs
func (p *Parser) ExtractURLs(reader io.Reader) ([]string, error) {
	var urls []string
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}

	// Extract URLs from HTML nodes
	var extractNode func(*html.Node)
	extractNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				if href := getAttr(n, "href"); href != "" {
					urls = append(urls, p.normalizeURL(href))
				}
			case "script":
				if src := getAttr(n, "src"); src != "" {
					urls = append(urls, p.normalizeURL(src))
				}
			case "link":
				if href := getAttr(n, "href"); href != "" {
					urls = append(urls, p.normalizeURL(href))
				}
			case "img":
				if src := getAttr(n, "src"); src != "" {
					urls = append(urls, p.normalizeURL(src))
				}
			case "form":
				if action := getAttr(n, "action"); action != "" {
					urls = append(urls, p.normalizeURL(action))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractNode(c)
		}
	}
	extractNode(doc)

	return urls, nil
}

// ExtractCSSURLs extracts URLs from CSS content
func (p *Parser) ExtractCSSURLs(content string) []string {
	var urls []string
	// Match url() patterns in CSS
	re := regexp.MustCompile(`url\(['"]?([^'"()]+)['"]?\)`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			urls = append(urls, p.normalizeURL(match[1]))
		}
	}
	return urls
}

// ExtractJSURLs extracts URLs from JavaScript content
func (p *Parser) ExtractJSURLs(content string) []string {
	var urls []string
	// Match fetch() and XHR patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`fetch\(['"]([^'"]+)['"]\)`),
		regexp.MustCompile(`\.open\(['"](?:GET|POST)['"],\s*['"]([^'"]+)['"]`),
		regexp.MustCompile(`\.get\(['"]([^'"]+)['"]`),
		regexp.MustCompile(`\.post\(['"]([^'"]+)['"]`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				urls = append(urls, p.normalizeURL(match[1]))
			}
		}
	}
	return urls
}

// normalizeURL converts a relative URL to an absolute URL
func (p *Parser) normalizeURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	// Handle protocol-relative URLs
	if strings.HasPrefix(urlStr, "//") {
		urlStr = p.baseURL.Scheme + ":" + urlStr
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	// If the URL is relative, resolve it against the base URL
	if !parsedURL.IsAbs() {
		parsedURL = p.baseURL.ResolveReference(parsedURL)
	}

	// Clean the URL
	parsedURL.Fragment = "" // Remove fragments
	return parsedURL.String()
}

// getAttr extracts an attribute value from an HTML node
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
} 