package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}
	defaultProfile := filepath.Join(homeDir, "snap/chromium/common/chromium")
	defaultDownload := filepath.Join(homeDir, "Downloads")

	profile := flag.String("profile", defaultProfile, "Path to Chromium profile")
	batchSize := flag.Int("batch", 10, "Number of dates per batch")
	execPath := flag.String("exec", "chromium", "Browser executable")
	downloadDir := flag.String("download", defaultDownload, "Directory to save downloads")
	flag.Parse()

	// Expand ~ in download path
	downloadPath := *downloadDir
	if strings.HasPrefix(downloadPath, "~/") {
		downloadPath = filepath.Join(homeDir, downloadPath[2:])
	}

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		log.Fatalf("Error creating download directory: %v", err)
	}

	log.Println("=== Yandex Photo Downloader ===")
	log.Printf("Executable: %s", *execPath)
	log.Printf("Profile: %s", *profile)
	log.Printf("Download: %s", downloadPath)
	log.Printf("Batch: %d dates at a time", *batchSize)

	if err := run(*profile, *batchSize, *execPath, downloadPath); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(profile string, batchSize int, execPath string, downloadDir string) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(execPath),
		chromedp.UserDataDir(profile),
		chromedp.Flag("headless", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1920, 1080),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()

	// 1. Open page
	log.Println("Opening Yandex Disk Photos...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://disk.yandex.com/client/photo"),
		chromedp.Sleep(5*time.Second),
	); err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}

	// Configure download directory
	if err := chromedp.Run(ctx,
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
			WithDownloadPath(downloadDir).
			WithEventsEnabled(true),
	); err != nil {
		log.Printf("⚠️ Warning: could not configure download directory: %v", err)
	} else {
		log.Printf("✓ Downloads will be saved to: %s", downloadDir)
	}

	// 2. Check login status
	isLoggedIn, err := checkLoginStatus(ctx)
	if err != nil {
		log.Printf("Warning: could not check login status: %v", err)
	}

	if !isLoggedIn {
		log.Println("⚠️  User is NOT logged in!")
		log.Println("⚠️  Please log in to your Yandex account in the browser window.")
		log.Println("Waiting for login (checking every 10 seconds, max 5 minutes)...")

		// Wait for user to login with periodic checks
		loginTimeout := time.After(5 * time.Minute)
		loginCheck := time.NewTicker(10 * time.Second)
		defer loginCheck.Stop()

		loginSuccess := false
		for !loginSuccess {
			select {
			case <-loginTimeout:
				return fmt.Errorf("login timeout: user did not log in within 5 minutes")
			case <-loginCheck.C:
				isLoggedIn, err = checkLoginStatus(ctx)
				if err != nil {
					log.Printf("Warning: login check failed: %v", err)
					continue
				}
				if isLoggedIn {
					loginSuccess = true
					log.Println("✓ Login detected!")
				} else {
					log.Println("Still waiting for login...")
				}
			}
		}

		// Navigate to photos after successful login
		if err := chromedp.Run(ctx,
			chromedp.Navigate("https://disk.yandex.com/client/photo"),
			chromedp.Sleep(5*time.Second),
		); err != nil {
			log.Printf("Warning: could not navigate after login: %v", err)
		}
	}

	log.Println("✓ User is logged in")

	// 3. Main loop - process one date at a time
	// Strategy: always process the FIRST visible date and scroll after each download
	totalProcessed := 0
	emptyRounds := 0

	for {
		log.Printf("\n--- Processing date %d ---", totalProcessed+1)

		// Check for pending selection and clear it
		if hasActiveSelection(ctx) {
			log.Println("⚠️ Pending selection detected, clearing...")
			if err := clickDeselect(ctx); err != nil {
				log.Printf("Warning: could not clear pending selection: %v", err)
			}
			time.Sleep(1 * time.Second)
		}

		// Select the FIRST visible date (always the top one)
		selected, dateText, yPosition, err := selectFirstVisibleDate(ctx)
		if err != nil {
			log.Printf("Error selecting: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if !selected {
			log.Println("No date found, scrolling...")
			if err := scrollDown(ctx); err != nil {
				log.Printf("Warning: scroll failed: %v", err)
			}
			time.Sleep(3 * time.Second)

			emptyRounds++
			if emptyRounds >= 5 {
				log.Println("End of photos!")
				break
			}
			continue
		}

		emptyRounds = 0
		log.Println("✓ Date selected: " + dateText)

		// Click Download
		time.Sleep(1500 * time.Millisecond)
		if err := clickDownload(ctx); err != nil {
			log.Printf("Download error: %v", err)
		} else {
			log.Println("✓ Download started")
		}

		// Wait for download to start
		time.Sleep(4 * time.Second)

		// Deselect
		for retry := 0; retry < 3; retry++ {
			if err := clickDeselect(ctx); err != nil {
				log.Printf("Error deselecting (attempt %d): %v", retry+1, err)
			}
			time.Sleep(1 * time.Second)

			if !hasActiveSelection(ctx) {
				break
			}
			log.Printf("⚠️ Selection still active, trying again...")
		}
		log.Println("✓ Deselected")

		// IMPORTANT: Scroll to move processed date off screen
		// This ensures we never process the same date twice
		if err := scrollToPosition(ctx, yPosition); err != nil {
			log.Printf("Warning: scroll to position failed: %v", err)
		}
		time.Sleep(1 * time.Second)

		totalProcessed++
	}

	log.Println("\n==================================================")
	log.Printf("✓ COMPLETED! %d dates processed", totalProcessed)
	log.Println("==================================================")
	log.Println("\nBrowser remains open. Press Ctrl+C to exit.")

	select {}
}

// selectFirstVisibleDate selects the FIRST visible date on screen
// Returns: (selected, dateText, yPosition, error)
func selectFirstVisibleDate(ctx context.Context) (bool, string, float64, error) {
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
		return false, "", 0, fmt.Errorf("error fetching dates: %w", err)
	}

	if dateInfo == nil {
		return false, "", 0, nil
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
		return false, "", 0, fmt.Errorf("error moving mouse: %w", err)
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
		return false, "", 0, fmt.Errorf("error clicking checkbox: %w", err)
	}

	if clicked {
		log.Printf("✓ Date '%s' selected", text)
		time.Sleep(500 * time.Millisecond)
		return true, text, y, nil
	}

	// Fallback: click directly
	err = chromedp.Run(ctx,
		chromedp.MouseClickXY(hoverX, y, chromedp.ButtonLeft),
	)
	if err == nil {
		log.Printf("✓ Date '%s' selected (direct click)", text)
		time.Sleep(500 * time.Millisecond)
		return true, text, y, nil
	}

	return false, "", 0, nil
}

// scrollToPosition scrolls to move the processed date off screen
func scrollToPosition(ctx context.Context, yPosition float64) error {
	// Scroll so the date is above the top of the screen (±300px)
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`window.scrollBy(0, %f - 50)`, yPosition), nil),
	); err != nil {
		return fmt.Errorf("scroll failed: %w", err)
	}
	log.Printf("Scroll executed to move date (y=%.0f) off screen", yPosition)
	return nil
}

// hasActiveSelection checks if there is any active selection on the page
func hasActiveSelection(ctx context.Context) bool {
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

func clickDownload(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const buttons = document.querySelectorAll('button, [role="button"]');
				for (const btn of buttons) {
					const text = btn.textContent?.trim() || '';
					const ariaLabel = btn.getAttribute('aria-label') || '';
					const title = btn.getAttribute('title') || '';
					
					if (text === 'Download' || 
						text === 'Скачать' ||
						ariaLabel.includes('Download') ||
						ariaLabel.includes('Скачать') ||
						title.includes('Download')) {
						btn.click();
						return 'clicked';
					}
				}
				return 'not found';
			})()
		`, nil),
	)
}

func clickDeselect(ctx context.Context) error {
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

func scrollDown(ctx context.Context) error {
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`window.scrollBy(0, 600)`, nil),
	); err != nil {
		return fmt.Errorf("scroll down failed: %w", err)
	}
	return nil
}

// checkLoginStatus verifies if the user is logged into Yandex
// Returns true if logged in, false if on login page
func checkLoginStatus(ctx context.Context) (bool, error) {
	// First check URL
	var url string
	if err := chromedp.Run(ctx, chromedp.Location(&url)); err != nil {
		return false, fmt.Errorf("could not get current URL: %w", err)
	}

	// If URL contains passport or auth, definitely not logged in
	if strings.Contains(url, "passport") || strings.Contains(url, "auth") {
		log.Printf("Login page detected (URL: %s)", url)
		return false, nil
	}

	// Check for login page elements in the DOM
	var isLoginPage bool
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				// Check for Yandex ID login page elements
				const pageText = document.body?.innerText || '';
				const pageHTML = document.body?.innerHTML || '';
				
				// Login page indicators
				const loginIndicators = [
					// Text content checks
					pageText.includes('Log in with Yandex ID'),
					pageText.includes('Войти с Яндекс ID'),
					pageText.includes('Yandex ID'),
					pageText.includes('Username or email'),
					pageText.includes('Логин или email'),
					pageText.includes('Create ID'),
					pageText.includes('Создать ID'),
					pageText.includes('Face or fingerprint login'),
					
					// Element checks
					!!document.querySelector('input[name="login"]'),
					!!document.querySelector('input[placeholder*="Username"]'),
					!!document.querySelector('input[placeholder*="email"]'),
					!!document.querySelector('button[data-t="button:pseudo"]'),
					!!document.querySelector('[class*="AuthLoginInputToggle"]'),
					!!document.querySelector('[class*="Passport"]'),
					!!document.querySelector('[data-t="login"]'),
					
					// Login form check
					!!document.querySelector('form[action*="passport"]'),
					!!document.querySelector('form[action*="auth"]'),
				];
				
				// If any login indicator is found, user is on login page
				return loginIndicators.some(indicator => indicator === true);
			})()
		`, &isLoginPage),
	)

	if err != nil {
		return false, fmt.Errorf("could not check login elements: %w", err)
	}

	if isLoginPage {
		log.Println("Login page elements detected in DOM")
		return false, nil
	}

	// Additional check: verify Yandex Disk elements are present (indicates logged in)
	var hasDiskElements bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				// Check for Yandex Disk logged-in elements
				const diskIndicators = [
					// Photo section elements
					!!document.querySelector('[class*="photo"]'),
					!!document.querySelector('[class*="Photo"]'),
					!!document.querySelector('[class*="listing"]'),
					!!document.querySelector('[class*="Listing"]'),
					
					// User avatar or account elements
					!!document.querySelector('[class*="user"]'),
					!!document.querySelector('[class*="User"]'),
					!!document.querySelector('[class*="avatar"]'),
					!!document.querySelector('[class*="Avatar"]'),
					
					// Disk navigation elements
					!!document.querySelector('[class*="sidebar"]'),
					!!document.querySelector('[class*="Sidebar"]'),
					!!document.querySelector('[href*="/client/"]'),
				];
				
				return diskIndicators.filter(i => i === true).length >= 2;
			})()
		`, &hasDiskElements),
	)

	if err != nil {
		log.Printf("Warning: could not verify disk elements: %v", err)
		// If we can't verify but URL looks OK, assume logged in
		return true, nil
	}

	if hasDiskElements {
		log.Println("✓ Yandex Disk elements detected - user is logged in")
		return true, nil
	}

	// If no disk elements found but also no login elements, wait a bit and recheck
	log.Println("⚠️ Could not confirm login status, page may still be loading...")
	return false, nil
}
