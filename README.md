# GoCreeper

GoCreeper is an intelligent web content discovery tool written in Go. Unlike traditional directory bruteforcing tools, GoCreeper uses smart crawling techniques to discover content by analyzing HTML, CSS, and JavaScript files.

## Features

- 🔍 Intelligent crawling of web content
- 🌐 Custom HTTP client with configurable headers and timeouts
- 📝 HTML parsing for links, scripts, stylesheets, and forms
- 🎨 CSS and JavaScript analysis
- 🔄 Queue management with depth control
- 🎯 Scope management to stay within target domain
- ⚡ Concurrent crawling with rate limiting
- 📊 Multiple output formats (tree, JSON, CSV)
- 🤖 Optional sitemap.xml and robots.txt parsing

## Installation

```bash
go install github.com/hunter/gocreeper@latest
```

## Usage

Basic usage:
```bash
gocreeper -u https://example.com
```

Advanced options:
```bash
gocreeper -u https://example.com \
    -d 3 \                    # Max depth
    -c 10 \                   # Concurrent workers
    -r 100ms \               # Request delay
    -o json \                # Output format (json/csv/tree)
    -H "User-Agent: Custom" \ # Custom headers
    -t 30s                    # Timeout
```

## Building from Source

```bash
git clone https://github.com/hunter/gocreeper
cd gocreeper
go build
```

## Project Structure

```
gocreeper/
├── main.go           # CLI entry point
├── crawler/
│   ├── crawler.go    # Core crawling logic
│   ├── parser.go     # HTML/CSS/JS parsing
│   └── client.go     # Custom HTTP client
└── go.mod           # Go module file
```

## License

MIT License - see LICENSE file for details 