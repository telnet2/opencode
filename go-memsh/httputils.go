package memsh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/itchyny/gojq"
)

// cmdJq implements the jq command for JSON processing
func (s *Shell) cmdJq(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("jq: missing filter expression")
	}

	// Parse flags first
	var input io.Reader
	files := []string{}
	rawOutput := false
	compact := false
	var filter string
	filterFound := false

	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "-r" || arg == "--raw-output" {
			rawOutput = true
			i++
		} else if arg == "-c" || arg == "--compact-output" {
			compact = true
			i++
		} else if !filterFound {
			// First non-flag argument is the filter
			filter = arg
			filterFound = true
			i++
		} else {
			// Remaining arguments are files
			files = append(files, arg)
			i++
		}
	}

	if !filterFound {
		return fmt.Errorf("jq: missing filter expression")
	}

	// If no files specified, read from stdin
	if len(files) == 0 {
		input = s.stdin
	} else {
		// Read from first file
		path := s.resolvePath(files[0])
		file, err := s.fs.Open(path)
		if err != nil {
			return fmt.Errorf("jq: %v", err)
		}
		defer file.Close()
		input = file
	}

	// Read and parse JSON input
	data, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("jq: read error: %v", err)
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("jq: parse error: %v", err)
	}

	// Parse and compile jq query
	query, err := gojq.Parse(filter)
	if err != nil {
		return fmt.Errorf("jq: filter parse error: %v", err)
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return fmt.Errorf("jq: compile error: %v", err)
	}

	// Execute query
	iter := code.Run(jsonData)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return fmt.Errorf("jq: execution error: %v", err)
		}

		// Format output
		if rawOutput {
			// Raw output mode - output strings without quotes
			switch val := v.(type) {
			case string:
				fmt.Fprintln(s.stdout, val)
			case nil:
				// null values produce no output in raw mode
			default:
				// Non-string values still get JSON encoding
				output, _ := json.Marshal(val)
				fmt.Fprintln(s.stdout, string(output))
			}
		} else if compact {
			// Compact output
			output, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("jq: marshal error: %v", err)
			}
			fmt.Fprintln(s.stdout, string(output))
		} else {
			// Pretty print by default
			output, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Errorf("jq: marshal error: %v", err)
			}
			fmt.Fprintln(s.stdout, string(output))
		}
	}

	return nil
}

// cmdCurl implements the curl command for HTTP requests
func (s *Shell) cmdCurl(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("curl: missing URL")
	}

	var method string = "GET"
	var url string
	var data string
	var headers []string
	var output string
	silent := false
	followRedirects := true
	includeHeaders := false

	// Parse arguments
	i := 1
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-X", "--request":
			if i+1 >= len(args) {
				return fmt.Errorf("curl: -X requires an argument")
			}
			method = args[i+1]
			i += 2
		case "-d", "--data":
			if i+1 >= len(args) {
				return fmt.Errorf("curl: -d requires an argument")
			}
			data = args[i+1]
			if method == "GET" {
				method = "POST"
			}
			i += 2
		case "-H", "--header":
			if i+1 >= len(args) {
				return fmt.Errorf("curl: -H requires an argument")
			}
			headers = append(headers, args[i+1])
			i += 2
		case "-o", "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("curl: -o requires an argument")
			}
			output = args[i+1]
			i += 2
		case "-s", "--silent":
			silent = true
			i++
		case "-L", "--location":
			followRedirects = true
			i++
		case "-i", "--include":
			includeHeaders = true
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("curl: unknown option: %s", arg)
			}
			url = arg
			i++
		}
	}

	if url == "" {
		return fmt.Errorf("curl: no URL specified")
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Create request
	var body io.Reader
	if data != "" {
		body = strings.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("curl: failed to create request: %v", err)
	}

	// Add headers
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	// Set default content type for POST with data
	if data != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Perform request
	if !silent {
		fmt.Fprintf(s.stderr, "* Requesting %s %s\n", method, url)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("curl: request failed: %v", err)
	}
	defer resp.Body.Close()

	if !silent {
		fmt.Fprintf(s.stderr, "* Response: %s\n", resp.Status)
	}

	// Determine output destination
	var writer io.Writer
	if output != "" {
		outputPath := s.resolvePath(output)
		file, err := s.fs.Create(outputPath)
		if err != nil {
			return fmt.Errorf("curl: failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	} else {
		writer = s.stdout
	}

	// Write headers if requested
	if includeHeaders {
		fmt.Fprintf(writer, "%s %s\n", resp.Proto, resp.Status)
		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Fprintf(writer, "%s: %s\n", key, value)
			}
		}
		fmt.Fprintln(writer)
	}

	// Write response body
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("curl: failed to read response: %v", err)
	}

	return nil
}
