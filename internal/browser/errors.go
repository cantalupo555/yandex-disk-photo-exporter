// Package browser provides Chrome/Chromedp initialization and configuration.
package browser

import (
	"context"
	"strings"
)

// IsBrowserClosed checks if an error indicates the browser was forcefully closed.
// This includes context canceled, context deadline exceeded, and common chromedp errors.
func IsBrowserClosed(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Check for context errors (most common when browser is closed)
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}

	// Check error message for common patterns
	closedPatterns := []string{
		"context canceled",
		"context deadline exceeded",
		"websocket: close",
		"target closed",
		"browser: not connected",
		"session closed",
		"page closed",
		"connection refused",
		"broken pipe",
	}

	for _, pattern := range closedPatterns {
		if strings.Contains(strings.ToLower(errMsg), pattern) {
			return true
		}
	}

	return false
}

// IsContextCanceled checks if the context is still valid.
func IsContextCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
