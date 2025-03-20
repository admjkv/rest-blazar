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
	outputFile := flag.String("save", "", "Save response body to file")
	bodyFile := flag.String("body-file", "", "File containing the request body")
	username := flag.String("user", "", "Username for basic auth")
	password := flag.String("pass", "", "Password for basic auth")
	verbose := flag.Bool("verbose", false, "Show request details")
	noRedirect := flag.Bool("no-redirect", false, "Don't follow redirects")
	http2 := flag.Bool("http2", false, "Force HTTP/2 protocol")
	jsonData := flag.String("json", "", "JSON data as key=value pairs (e.g. name=John,age=30)")
	formData := flag.String("form", "", "Form data as key=value pairs (e.g. name=John,age=30)")
	retries := flag.Int("retries", 0, "Number of retry attempts for failed requests")
	retryDelay := flag.Int("retry-delay", 1, "Delay between retries in seconds")
	flag.Parse()

	// check for url
	if *url == "" {
		fmt.Println("Error: URL is required.")
		os.Exit(1)
	}

	// create http client with custom settings
	client := http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
	}

	// Configure HTTP/2 transport if requested
	if *http2 {
		transport := &http.Transport{
			ForceAttemptHTTP2: true,
		}
		client.Transport = transport
	}

	// configure redirect policy
	if *noRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// determine the request body
	var reqBody io.Reader
	if *bodyFile != "" {
		fileData, err := os.ReadFile(*bodyFile)
		if err != nil {
			fmt.Printf("Error reading body file: %v\n", err)
			os.Exit(1)
		}
		reqBody = strings.NewReader(string(fileData))
	} else if *jsonData != "" {
		// Process JSON data from command line
		jsonMap := make(map[string]interface{})
		pairs := strings.Split(*jsonData, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				jsonMap[parts[0]] = parts[1]
			}
		}
		jsonBytes, err := json.Marshal(jsonMap)
		if err != nil {
			fmt.Printf("Error creating JSON: %v\n", err)
			os.Exit(1)
		}
		reqBody = strings.NewReader(string(jsonBytes))
		// Set JSON content type if not overridden
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	} else if *formData != "" {
		// Process form data
		formValues := url.Values{}
		pairs := strings.Split(*formData, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				formValues.Add(parts[0], parts[1])
			}
		}
		reqBody = strings.NewReader(formValues.Encode())
		// Set form content type if not overridden
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		reqBody = strings.NewReader(*body)
	}

	// build the request
	req, err := http.NewRequest(*method, *url, reqBody)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	if *username != "" {
		req.SetBasicAuth(*username, *password)
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

	// display request information in verbose mode
	if *verbose {
		fmt.Printf("\n> %s %s\n", req.Method, req.URL)
		for key, values := range req.Header {
			fmt.Printf("> %s: %s\n", key, strings.Join(values, ", "))
		}
		if *body != "" || *bodyFile != "" {
			fmt.Println("> ")
			fmt.Println("> " + *body)
		}
		fmt.Println()
	}

	// Implement retries
	var respData []byte
	var finalResp *http.Response
	var finalErr error

	for attempt := 0; attempt <= *retries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retry attempt %d/%d...\n", attempt, *retries)
			time.Sleep(time.Duration(*retryDelay) * time.Second)
		}

		startTime := time.Now()
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err == nil {
				duration = time.Since(startTime)
				respData = data
				finalResp = resp
				finalErr = nil
				break
			}
			finalErr = err
		} else {
			finalErr = err
		}
	}

	if finalErr != nil {
		fmt.Printf("Error after %d attempts: %v\n", *retries+1, finalErr)
		os.Exit(1)
	}

	resp = finalResp
	data := respData

	duration := time.Since(startTime)

	// Display timing stats in verbose mode
	if *verbose {
		fmt.Printf("\nRequest completed in %v\n", duration)
		fmt.Printf("Time to first byte: %v\n", time.Since(startTime))
	}

	// response output
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		os.Exit(1)
	}

	if *outputFile != "" {
		err := os.WriteFile(*outputFile, data, 0644)
		if err != nil {
			fmt.Printf("Error saving response to file: %v\n", err)
		} else {
			fmt.Printf("Response saved to %s\n", *outputFile)
		}
	}

	switch *output {
	case "json":
		outputJSON(resp, data, duration)
	case "headers-only":
		outputHeaders(resp)
	case "body-only":
		fmt.Println(string(data))
	default: // "pretty"
		outputPretty(resp, data, duration)
	}
}

func outputPretty(resp *http.Response, data []byte, duration time.Duration) {
	// color codes for status
	var statusColor string
	switch {
	case resp.StatusCode >= 500:
		statusColor = "\033[31m" // Red for 5xx
	case resp.StatusCode >= 400:
		statusColor = "\033[33m" // Yellow for 4xx
	case resp.StatusCode >= 300:
		statusColor = "\033[36m" // Cyan for 3xx
	case resp.StatusCode >= 200:
		statusColor = "\033[32m" // Green for 2xx
	default:
		statusColor = "\033[37m" // White for other
	}
	resetColor := "\033[0m"

	fmt.Printf("Status: %s%s%s\n", statusColor, resp.Status, resetColor)
	fmt.Println("Headers:")
	for key, values := range resp.Header {
		fmt.Printf("  %s: %s\n", key, strings.Join(values, ", "))
	}
	fmt.Println("Body:")
	fmt.Println(string(data))
	fmt.Printf("Request completed in %v\n", duration)
}

func outputJSON(resp *http.Response, data []byte, duration time.Duration) {
	result := map[string]interface{}{
		"status":     resp.Status,
		"statusCode": resp.StatusCode,
		"headers":    resp.Header,
		"body":       string(data),
		"duration":   duration.String(),
	}
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON response: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

func outputHeaders(resp *http.Response) {
	for key, values := range resp.Header {
		fmt.Printf("%s: %s\n", key, strings.Join(values, ", "))
	}
}
