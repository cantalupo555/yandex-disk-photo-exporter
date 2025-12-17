// Package selection handles photo date selection and deselection on Yandex Disk.
package selection

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// DateInfo contains information about a selected date.
type DateInfo struct {
	Text      string
	YPosition float64
}

// SelectFirstVisibleDate selects the FIRST visible date on screen.
// Returns the date info if selected, nil if no date found.
func SelectFirstVisibleDate(ctx context.Context) (*DateInfo, error) {
	// Get the first visible date
	var dateInfo map[string]interface{}
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const allElements = document.querySelectorAll('*');
				const dates = [];
				
				allElements.forEach(el => {
					const text = el.textContent?.trim() || '';
					// Detect date pattern
					if (/^\d{1,2}\s+(January|February|March|April|May|June|July|August|September|October|November|December)$/i.test(text)) {
						const rect = el.getBoundingClientRect();
						// Only include if visible on screen
						if (rect.top >= 80 && rect.top < window.innerHeight - 50 && rect.width > 0) {
							dates.push({
								text: text,
								x: rect.left,
								y: rect.top + (rect.height / 2)
							});
						}
					}
				});
				
				// Sort by Y and return the first one
				dates.sort((a, b) => a.y - b.y);
				return dates.length > 0 ? dates[0] : null;
			})()
		`, &dateInfo),
	)

	if err != nil {
		return nil, fmt.Errorf("error fetching dates: %w", err)
	}

	if dateInfo == nil {
		return nil, nil
	}

	x, _ := dateInfo["x"].(float64)
	y, _ := dateInfo["y"].(float64)
	text, _ := dateInfo["text"].(string)

	log.Printf("Processing FIRST visible date: %s (y=%.0f)", text, y)

	// Hover on left side to reveal checkbox
	hoverX := x - 30
	if hoverX < 10 {
		hoverX = 10
	}

	err = chromedp.Run(ctx,
		chromedp.MouseClickXY(hoverX, y, chromedp.ButtonNone),
	)
	if err != nil {
		return nil, fmt.Errorf("error moving mouse: %w", err)
	}

	time.Sleep(2 * time.Second)

	// Click on checkbox
	var clicked bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`
			(function() {
				const targetY = %f;
				const checkboxes = document.querySelectorAll('input[type="checkbox"], [class*="checkbox"], [class*="Checkbox"]');
				
				for (const cb of checkboxes) {
					const rect = cb.getBoundingClientRect();
					if (Math.abs(rect.top + rect.height/2 - targetY) < 40) {
						if (!cb.checked && !cb.classList.contains('checked')) {
							cb.click();
							return true;
						}
					}
				}
				
				const elements = document.elementsFromPoint(%f, %f);
				for (const el of elements) {
					if (el.tagName === 'INPUT' || 
						el.className?.includes('checkbox') || 
						el.className?.includes('Checkbox') ||
						el.role === 'checkbox') {
						el.click();
						return true;
					}
				}
				
				return false;
			})()
		`, y, hoverX, y), &clicked),
	)

	if err != nil {
		return nil, fmt.Errorf("error clicking checkbox: %w", err)
	}

	if clicked {
		log.Printf("✓ Date '%s' selected", text)
		time.Sleep(500 * time.Millisecond)
		return &DateInfo{Text: text, YPosition: y}, nil
	}

	// Fallback: click directly
	err = chromedp.Run(ctx,
		chromedp.MouseClickXY(hoverX, y, chromedp.ButtonLeft),
	)
	if err == nil {
		log.Printf("✓ Date '%s' selected (direct click)", text)
		time.Sleep(500 * time.Millisecond)
		return &DateInfo{Text: text, YPosition: y}, nil
	}

	return nil, nil
}

// HasActiveSelection checks if there is any active selection on the page.
func HasActiveSelection(ctx context.Context) bool {
	var hasSelection bool
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				// Check if selection bar is visible (file counter)
				const selectionBar = document.querySelector('[class*="selection"], [class*="toolbar"]');
				if (selectionBar) {
					const text = selectionBar.textContent || '';
					if (/\d+\s*(file|файл|item)/i.test(text)) {
						return true;
					}
				}
				
				// Check if there are checked checkboxes
				const checkedInputs = document.querySelectorAll('input[type="checkbox"]:checked');
				if (checkedInputs.length > 0) return true;
				
				// Check elements with 'checked' class
				const checkedElements = document.querySelectorAll('[class*="checkbox"][class*="checked"]');
				if (checkedElements.length > 0) return true;
				
				return false;
			})()
		`, &hasSelection),
	); err != nil {
		log.Printf("Warning: could not check selection state: %v", err)
		return false
	}
	return hasSelection
}

// Deselect clears the current selection by clicking the X button or pressing ESC.
func Deselect(ctx context.Context) error {
	// Find the X button (close/deselect) in the selection bar
	var buttonInfo map[string]interface{}
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				// Look for X or Deselect button in top bar
				const selectors = [
					'button[aria-label*="close" i]',
					'button[aria-label*="Close" i]',
					'button[aria-label*="deselect" i]',
					'[class*="close"]',
					'[class*="Close"]',
					'svg[class*="close"]',
					'button:has(svg)',
				];
				
				for (const selector of selectors) {
					const elements = document.querySelectorAll(selector);
					for (const el of elements) {
						const rect = el.getBoundingClientRect();
						// X button should be at the top of the screen (toolbar)
						if (rect.top < 150 && rect.width > 0 && rect.height > 0) {
							const text = el.textContent?.trim() || '';
							const ariaLabel = el.getAttribute('aria-label') || '';
							// Check if it looks like a close/deselect button
							if (text === '×' || text === 'X' || text === '' || 
								ariaLabel.toLowerCase().includes('close') ||
								ariaLabel.toLowerCase().includes('deselect')) {
								return {
									x: rect.left + rect.width/2,
									y: rect.top + rect.height/2,
									found: true,
									info: ariaLabel || text || 'button'
								};
							}
						}
					}
				}
				
				// Look for any button in top bar that could be the X
				const allButtons = document.querySelectorAll('button, [role="button"]');
				for (const btn of allButtons) {
					const rect = btn.getBoundingClientRect();
					// Button in top right corner (selection area)
					if (rect.top < 100 && rect.right > window.innerWidth - 200) {
						const text = btn.textContent?.trim() || '';
						if (text === '×' || text === 'X' || text.length <= 2) {
							return {
								x: rect.left + rect.width/2,
								y: rect.top + rect.height/2,
								found: true,
								info: 'corner-button'
							};
						}
					}
				}
				
				return { found: false };
			})()
		`, &buttonInfo),
	)

	if err != nil {
		return err
	}

	found, _ := buttonInfo["found"].(bool)
	if found {
		x, _ := buttonInfo["x"].(float64)
		y, _ := buttonInfo["y"].(float64)
		info, _ := buttonInfo["info"].(string)
		log.Printf("Clicking X button at (%.0f, %.0f) - %s", x, y, info)

		err = chromedp.Run(ctx,
			chromedp.MouseClickXY(x, y, chromedp.ButtonLeft),
		)
		if err != nil {
			log.Printf("Error clicking X: %v", err)
		}
	} else {
		log.Println("X button not found, trying ESC...")
		// Fallback: press ESC
		if err := chromedp.Run(ctx, chromedp.KeyEvent("\x1b")); err != nil {
			log.Printf("Warning: ESC key press failed: %v", err)
		}
	}

	// Wait for UI to update
	time.Sleep(1 * time.Second)

	// Check if selection is still active and click on empty area
	var hasSelection bool
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const checked = document.querySelectorAll('[class*="checkbox"][class*="checked"], [class*="selected"]');
				return checked.length > 0;
			})()
		`, &hasSelection),
	); err != nil {
		log.Printf("Warning: could not check remaining selection: %v", err)
	}

	if hasSelection {
		log.Println("Selection still active, clicking on empty area...")
		// Click on an empty area of the page
		if err := chromedp.Run(ctx, chromedp.MouseClickXY(800, 400, chromedp.ButtonLeft)); err != nil {
			log.Printf("Warning: click on empty area failed: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

// ClearPendingSelection checks and clears any pending selection.
func ClearPendingSelection(ctx context.Context) {
	if HasActiveSelection(ctx) {
		log.Println("⚠️ Pending selection detected, clearing...")
		if err := Deselect(ctx); err != nil {
			log.Printf("Warning: could not clear pending selection: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
}
