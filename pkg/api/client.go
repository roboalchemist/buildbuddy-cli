package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a BuildBuddy API client.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	debug      func(string, ...interface{})
}

// NewClient creates a new BuildBuddy API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// SetDebug sets a debug logging function.
func (c *Client) SetDebug(fn func(string, ...interface{})) {
	c.debug = fn
}

func (c *Client) debugLog(format string, args ...interface{}) {
	if c.debug != nil {
		c.debug(format, args...)
	}
}

// Call performs a POST request to the given BuildBuddy API endpoint.
func (c *Client) Call(endpoint string, body interface{}, result interface{}) error {
	url := fmt.Sprintf("%s/api/v1/%s", c.baseURL, endpoint)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	c.debugLog("POST %s", url)
	c.debugLog("Request body: %s", string(jsonBody))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-buildbuddy-api-key", c.apiKey)
	req.Header.Set("User-Agent", "buildbuddy-cli/"+version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	c.debugLog("Response status: %d", resp.StatusCode)
	c.debugLog("Response body: %s", truncate(string(respBody), 2000))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// PageTokenSetter is implemented by requests that support pagination.
type PageTokenSetter interface {
	SetPageToken(string)
}

const maxPages = 100

// CallAllPages calls an endpoint repeatedly, following next_page_token until exhausted.
// onPage is called after each page (including the first). The caller should extract and
// accumulate data from resp in the onPage callback, as resp gets overwritten each page.
func (c *Client) CallAllPages(endpoint string, req PageTokenSetter, resp Paginated, onPage func()) error {
	for page := 0; page < maxPages; page++ {
		if err := c.Call(endpoint, req, resp); err != nil {
			return err
		}
		onPage()

		token := resp.GetNextPageToken()
		if token == "" {
			break
		}
		c.debugLog("Paginating page %d: next_page_token=%s", page+1, truncate(token, 40))
		req.SetPageToken(token)
	}

	return nil
}

// version is injected at build time, used in User-Agent.
var version = "dev"

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
