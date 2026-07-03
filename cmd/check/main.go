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
	endpoint      = "https://jsonplaceholder.typicode.com/todos/1"
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

	value, body, err := checkEndpoint(client, endpoint, requiredField)
	if body != nil {
		fmt.Printf("Response: %s\n", string(body))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("OK: field %q is present with value: %v\n", requiredField, value)
}

// checkEndpoint GETs url and verifies that field is present in the JSON body.
// It returns the field's value and the raw response body (for logging). Any
// non-200 status, unreadable body, invalid JSON, or missing field is an error.
func checkEndpoint(client *http.Client, url, field string) (value any, body []byte, err error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("ERROR: failed to reach endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("ERROR: unexpected status code %d", resp.StatusCode)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("ERROR: failed to read response body: %w", err)
	}

	// Unmarshal into a generic map so we can detect a missing field.
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, body, fmt.Errorf("ERROR: invalid JSON: %w", err)
	}

	value, exists := raw[field]
	if !exists {
		return nil, body, fmt.Errorf("FAIL: required field %q is missing from the response", field)
	}

	return value, body, nil
}
