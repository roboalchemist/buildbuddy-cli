package auth

import (
	"fmt"
	"os"
)

// GetAPIKey returns the BuildBuddy API key from the environment.
func GetAPIKey() (string, error) {
	key := os.Getenv("BUILDBUDDY_API_KEY")
	if key == "" {
		return "", fmt.Errorf("BUILDBUDDY_API_KEY not set")
	}
	return key, nil
}

// GetBaseURL returns the BuildBuddy base URL, defaulting to the hosted instance.
func GetBaseURL() string {
	if url := os.Getenv("BUILDBUDDY_BASE_URL"); url != "" {
		return url
	}
	return "https://app.buildbuddy.io"
}
