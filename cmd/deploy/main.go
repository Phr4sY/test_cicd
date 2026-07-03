package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type DeployRequest struct {
	Version string `json:"version"`
	Service string `json:"service"`
}

type DeployResponse struct {
	Status   string `json:"status"`
	DeployID string `json:"deployId"`
	Message  string `json:"message"`
}

const serviceName = "test-cicd-app"

func main() {
	deployURL := requireEnv("DEPLOY_URL")
	deployToken := requireEnv("DEPLOY_TOKEN")
	appVersion := getEnv("APP_VERSION", "latest")

	fmt.Printf("Deploying service version %q to %s\n", appVersion, deployURL)

	client := &http.Client{Timeout: 30 * time.Second}

	msg, err := deploy(client, deployURL, deployToken, appVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Println(msg)
}

// deploy POSTs a deploy request to baseURL+"/deploy" and interprets the
// response. It returns a human-readable success message, or an error for a
// transport failure, a >=400 status, or a body reporting {"status":"failed"}.
func deploy(client *http.Client, baseURL, token, version string) (string, error) {
	payload := DeployRequest{Version: version, Service: serviceName}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal deploy request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/deploy", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("deploy request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read deploy response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("deploy returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result DeployResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Not all APIs return structured JSON — treat any 2xx as success.
		return fmt.Sprintf("OK: deploy succeeded (HTTP %d)", resp.StatusCode), nil
	}

	if result.Status == "failed" {
		return "", fmt.Errorf("deploy reported failure: %s", result.Message)
	}

	return fmt.Sprintf("OK: deploy succeeded — ID: %s, message: %s", result.DeployID, result.Message), nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("required environment variable %q is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
