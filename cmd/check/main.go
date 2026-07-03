package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	endpoint     = "https://jsonplaceholder.typicode.com/todos/1"
	requiredField = "completed"
)

type Todo struct {
	UserID    int    `json:"userId"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed *bool  `json:"completed"` // pointer so we can detect missing vs false
}

func main() {
	fmt.Printf("Checking endpoint: %s\n", endpoint)

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to reach endpoint: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "ERROR: unexpected status code %d\n", resp.StatusCode)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read response body: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response: %s\n", string(body))

	// Unmarshal into a generic map first to detect missing fields
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid JSON: %v\n", err)
		os.Exit(1)
	}

	value, exists := raw[requiredField]
	if !exists {
		fmt.Fprintf(os.Stderr, "FAIL: required field %q is missing from the response\n", requiredField)
		os.Exit(1)
	}

	fmt.Printf("OK: field %q is present with value: %v\n", requiredField, value)
}
