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
	Status    string `json:"status"`
	DeployID  string `json:"deployId"`
	Message   string `json:"message"`
}

func main() {
	deployURL := requireEnv("DEPLOY_URL")
	deployToken := requireEnv("DEPLOY_TOKEN")
	appVersion := getEnv("APP_VERSION", "latest")

	fmt.Printf("Deploying service version %q to %s\n", appVersion, deployURL)

	payload := DeployRequest{
		Version: appVersion,
		Service: "test-cicd-app",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		fatalf("failed to marshal deploy request: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodPost, deployURL+"/deploy", bytes.NewReader(body))
	if err != nil {
		fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+deployToken)

	resp, err := client.Do(req)
	if err != nil {
		fatalf("deploy request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fatalf("failed to read deploy response: %v", err)
	}

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "FAIL: deploy returned HTTP %d: %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	var result DeployResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Not all APIs return structured JSON — treat a 2xx as success.
		fmt.Printf("OK: deploy succeeded (HTTP %d)\n", resp.StatusCode)
		return
	}

	if result.Status == "failed" {
		fmt.Fprintf(os.Stderr, "FAIL: deploy reported failure: %s\n", result.Message)
		os.Exit(1)
	}

	fmt.Printf("OK: deploy succeeded — ID: %s, message: %s\n", result.DeployID, result.Message)
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
