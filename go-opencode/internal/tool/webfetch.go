package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	einotool "github.com/cloudwego/eino/components/tool"
)

const webfetchDescription = `Fetches content from a specified URL and returns it in the requested format.

Usage notes:
  - IMPORTANT: If an MCP-provided web fetch tool is available, prefer using that tool instead of this one, as it may have fewer restrictions.
  - The URL must be a fully-formed valid URL starting with http:// or https://
  - HTTP URLs will be automatically upgraded to HTTPS
  - This tool is read-only and does not modify any files
  - Results may be truncated if the content is very large (>5MB limit)
  - Use format "markdown" for readable content, "text" for plain text, "html" for raw HTML`

const (
	maxResponseSize = 5 * 1024 * 1024 // 5MB
	defaultTimeout  = 30 * time.Second
	maxTimeout      = 120 * time.Second
)

// WebFetchTool implements web content fetching.
type WebFetchTool struct {
	workDir string
	client  *http.Client
}

// WebFetchInput represents the input for the webfetch tool.
// SDK compatible: uses camelCase field names to match TypeScript.
type WebFetchInput struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Timeout int    `json:"timeout,omitempty"`
}

// NewWebFetchTool creates a new webfetch tool.
func NewWebFetchTool(workDir string) *WebFetchTool {
	return &WebFetchTool{
		workDir: workDir,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (t *WebFetchTool) ID() string          { return "webfetch" }
func (t *WebFetchTool) Description() string { return webfetchDescription }

func (t *WebFetchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "The URL to fetch content from"
			},
			"format": {
				"type": "string",
				"enum": ["text", "markdown", "html"],
				"description": "The format to return the content in (text, markdown, or html)"
			},
			"timeout": {
				"type": "integer",
				"description": "Optional timeout in seconds (max 120)"
			}
		},
		"required": ["url", "format"]
	}`)
}

func (t *WebFetchTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params WebFetchInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate URL
	if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
		return nil, fmt.Errorf("URL must start with http:// or https://")
	}

	// Validate format
	if params.Format != "text" && params.Format != "markdown" && params.Format != "html" {
		return nil, fmt.Errorf("format must be 'text', 'markdown', or 'html'")
	}

	// Calculate timeout
	timeout := defaultTimeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
		if timeout > maxTimeout {
			timeout = maxTimeout
		}
	}

	// Create HTTP request with context and timeout
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", params.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers based on format
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	switch params.Format {
	case "markdown":
		req.Header.Set("Accept", "text/markdown;q=1.0, text/x-markdown;q=0.9, text/plain;q=0.8, text/html;q=0.7, */*;q=0.1")
	case "text":
		req.Header.Set("Accept", "text/plain;q=1.0, text/markdown;q=0.9, text/html;q=0.8, */*;q=0.1")
	case "html":
		req.Header.Set("Accept", "text/html;q=1.0, application/xhtml+xml;q=0.9, text/plain;q=0.8, text/markdown;q=0.7, */*;q=0.1")
	default:
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	}

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	// Check content length header
	if resp.ContentLength > maxResponseSize {
		return nil, fmt.Errorf("response too large (exceeds 5MB limit)")
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("response too large (exceeds 5MB limit)")
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")
	title := fmt.Sprintf("%s (%s)", params.URL, contentType)

	// Process content based on format
	var output string
	switch params.Format {
	case "markdown":
		if strings.Contains(contentType, "text/html") {
			output, err = convertHTMLToMarkdown(content)
			if err != nil {
				return nil, fmt.Errorf("failed to convert HTML to markdown: %w", err)
			}
		} else {
			output = content
		}
	case "text":
		if strings.Contains(contentType, "text/html") {
			output, err = extractTextFromHTML(content)
			if err != nil {
				return nil, fmt.Errorf("failed to extract text from HTML: %w", err)
			}
		} else {
			output = content
		}
	case "html":
		output = content
	default:
		output = content
	}

	return &Result{
		Title:    title,
		Output:   output,
		Metadata: map[string]any{},
	}, nil
}

func (t *WebFetchTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}

// extractTextFromHTML extracts plain text from HTML, removing scripts, styles, and other non-content elements.
func extractTextFromHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Remove non-content elements
	doc.Find("script, style, noscript, iframe, object, embed").Remove()

	// Get text content
	text := doc.Text()

	// Clean up whitespace
	text = strings.TrimSpace(text)

	return text, nil
}

// convertHTMLToMarkdown converts HTML content to Markdown format.
func convertHTMLToMarkdown(html string) (string, error) {
	converter := md.NewConverter("", true, &md.Options{
		HeadingStyle:    "atx",
		HorizontalRule:  "---",
		BulletListMarker: "-",
		CodeBlockStyle:  "fenced",
		EmDelimiter:     "*",
	})

	// Remove non-content elements
	converter.Remove("script", "style", "meta", "link")

	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}

	return markdown, nil
}
