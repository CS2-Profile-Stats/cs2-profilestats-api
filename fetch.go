package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type APIError struct {
  StatusCode int
  Body       string
}

type APIErrorBody struct {
  Errors []APIErrorContent `json:"errors"`
}

type APIErrorContent struct {
  Message string `json:"message"`
}

func (e *APIError) Error() string {
  var body APIErrorBody
  if err := json.Unmarshal([]byte(e.Body), &body); err == nil && len(body.Errors) > 0 {
    return body.Errors[0].Message
  }
  return fmt.Sprintf("Error %d", e.StatusCode)
}

type Fetcher struct {
	apiKey     string
	authHeader string
	httpClient *http.Client
}

func newFetcher(apiKey, authHeader string) Fetcher {
	return Fetcher{
		apiKey:     apiKey,
		authHeader: authHeader,
		httpClient: &http.Client{},
	}
}

func (f *Fetcher) fetch(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed creating a request: %w", err)
	}

	// steam for example doesn't have a header and its passed as a url parameter
	if f.authHeader != "" {
		req.Header.Set(f.authHeader, f.apiKey)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed executing a request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return nil, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed reading body: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Failed parsing json: %w", err)
	}

	return result, nil
}
