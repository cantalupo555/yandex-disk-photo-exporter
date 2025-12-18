// Package navigation handles page scrolling and navigation on Yandex Disk.
package navigation

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// FilterByUnlimitedStorage clicks on the filter menu and selects "From unlimited storage"
// to filter photos that need to be downloaded.
func FilterByUnlimitedStorage(ctx context.Context) error {
	log.Println("Applying filter: From unlimited storage...")

	// Wait for page to be fully loaded
	time.Sleep(2 * time.Second)

	// Step 1: Click the filter menu button
	// The button has aria-label starting with "Show:" and class "Select2-Button"
	menuButtonSelector := `button.Select2-Button[aria-label^="Show:"]`

	err := chromedp.Run(ctx,
		chromedp.WaitVisible(menuButtonSelector, chromedp.ByQuery),
		chromedp.Click(menuButtonSelector, chromedp.ByQuery),
	)
	if err != nil {
		// Try alternative selector
		altSelector := `button[role="listbox"].Select2-Button`
		err = chromedp.Run(ctx,
			chromedp.WaitVisible(altSelector, chromedp.ByQuery),
			chromedp.Click(altSelector, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("could not click filter menu button: %w", err)
		}
	}
	log.Println("✓ Filter menu opened")

	// Wait for menu to appear
	time.Sleep(500 * time.Millisecond)

	// Step 2: Click "From unlimited storage" option
	// Use JavaScript to find and click the menu item by text content
	var clicked bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				// Find all menu items
				const menuItems = document.querySelectorAll('.Menu-Item[role="option"]');
				for (const item of menuItems) {
					if (item.textContent.includes('unlimited storage') || 
					    item.textContent.includes('Unlimited storage')) {
						item.click();
						return true;
					}
				}
				return false;
			})()
		`, &clicked),
	)

	if err != nil {
		return fmt.Errorf("error executing click on menu item: %w", err)
	}

	if !clicked {
		// Try XPath as fallback
		xpathSelector := `//div[@role="option"][contains(., "unlimited storage")]`
		err = chromedp.Run(ctx,
			chromedp.Click(xpathSelector, chromedp.BySearch),
		)
		if err != nil {
			return fmt.Errorf("could not find 'From unlimited storage' option: %w", err)
		}
	}

	log.Println("✓ 'From unlimited storage' filter selected")

	// Wait a moment for selection to register
	time.Sleep(300 * time.Millisecond)

	// Step 3: Close the menu by clicking the button again or clicking elsewhere
	err = chromedp.Run(ctx,
		chromedp.Click(menuButtonSelector, chromedp.ByQuery),
	)
	if err != nil {
		// If clicking button fails, try clicking elsewhere on the page to close menu
		chromedp.Run(ctx,
			chromedp.Evaluate(`document.body.click()`, nil),
		)
	}
	log.Println("✓ Filter menu closed")

	// Wait for filter to be applied and page to update
	time.Sleep(2 * time.Second)

	log.Println("✓ Filter applied successfully")
	return nil
}
