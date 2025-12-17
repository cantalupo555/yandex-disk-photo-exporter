// Package navigation handles page scrolling and navigation on Yandex Disk.
package navigation

import (
	"context"
	"fmt"
	"log"

	"github.com/chromedp/chromedp"
)

const (
	// DefaultScrollAmount is the default number of pixels to scroll down.
	DefaultScrollAmount = 600
)

// ScrollDown scrolls the page down by the default amount.
func ScrollDown(ctx context.Context) error {
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`window.scrollBy(0, %d)`, DefaultScrollAmount), nil),
	); err != nil {
		return fmt.Errorf("scroll down failed: %w", err)
	}
	return nil
}

// ScrollToPosition scrolls to move the processed date off screen.
func ScrollToPosition(ctx context.Context, yPosition float64) error {
	// Scroll so the date is above the top of the screen (±300px)
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`window.scrollBy(0, %f - 50)`, yPosition), nil),
	); err != nil {
		return fmt.Errorf("scroll failed: %w", err)
	}
	log.Printf("Scroll executed to move date (y=%.0f) off screen", yPosition)
	return nil
}
