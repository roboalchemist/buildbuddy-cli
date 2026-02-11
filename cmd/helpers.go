package cmd

import (
	"fmt"

	"github.com/roboalchemist/buildbuddy-cli/pkg/api"
	"github.com/roboalchemist/buildbuddy-cli/pkg/auth"
	"github.com/roboalchemist/buildbuddy-cli/pkg/output"
)

// newClient creates an authenticated BuildBuddy API client.
func newClient() (*api.Client, error) {
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return nil, output.NewAuthError(err.Error())
	}
	baseURL := auth.GetBaseURL()
	client := api.NewClient(baseURL, apiKey)
	if flagDebug {
		client.SetDebug(DebugLog)
	}
	return client, nil
}

// formatDuration converts microseconds to a human-readable duration string.
func formatDuration(usec int64) string {
	if usec <= 0 {
		return "-"
	}
	sec := usec / 1_000_000
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	min := sec / 60
	remSec := sec % 60
	if min < 60 {
		return fmt.Sprintf("%dm%02ds", min, remSec)
	}
	hr := min / 60
	remMin := min % 60
	return fmt.Sprintf("%dh%02dm", hr, remMin)
}

// truncateStr truncates a string to max length with ellipsis.
func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
