package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// command-line flags for customization
	method := flag.String("method", "GET", "HTTP method to use")
	url := flag.String("url", "", "URL to send request to")
	body := flag.String("body", "", "Body to send with request")
	headers := flag.String("headers", "", "Headers to send with request")
	timeout := flag.Int("timeout", 10, "Timeout in seconds")
	output := flag.String("output", "pretty", "Output format: pretty, json, headers-only, body-only")
	flag.Parse()

	// check for url
	if *url == "" {
		fmt.Println("Error: URL is required.")
		os.Exit(1)
	}

	// create http request with timout
	client := http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
	}

	// build the request
	req, err := http.NewRequest(*method, *url, strings.NewReader(*body))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	// add headers if provided
	if *headers != "" {
		pairs := strings.Split(*headers, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				req.Header.Set(key, value)
			}
		}
	} else {
		// default header fallback
		req.Header.Set("Content-Type", "application/json")
	}

	// send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error performing request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// response output
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		os.Exit(1)
	}

	switch *output {
	case "json":
		outputJSON(resp, data)
	case "headers-only":
		outputHeaders(resp)
	case "body-only":
		fmt.Println(string(data))
	default: // "pretty"
		outputPretty(resp, data)
	}
}

func outputPretty(resp *http.Response, data []byte) {
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Println("Headers:")
	for key, values := range resp.Header {
		fmt.Printf("  %s: %s\n", key, strings.Join(values, ", "))
	}
	fmt.Println("Body:")
	fmt.Println(string(data))
}

func outputJSON(resp *http.Response, data []byte) {
	result := map[string]interface{}{
		"status":     resp.Status,
		"statusCode": resp.StatusCode,
		"headers":    resp.Header,
		"body":       string(data),
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}

func outputHeaders(resp *http.Response) {
	for key, values := range resp.Header {
		fmt.Printf("%s: %s\n", key, strings.Join(values, ", "))
	}
}
