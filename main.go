package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/auth"
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/browser"
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/datefilter"
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/download"
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/navigation"
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/report"
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/selection"
)

// appVersion is set at build time via -ldflags="-X main.appVersion=x.x.x"
var appVersion = "dev"

const (
	yandexPhotosURL = "https://disk.yandex.com/client/photo"
)

func main() {
	// Version flag
	showVersion := flag.Bool("version", false, "Show version and exit")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}

	// OS-aware defaults
	defaultProfile := browser.DefaultProfilePath()
	defaultDownload := filepath.Join(homeDir, "Downloads")

	profile := flag.String("profile", defaultProfile, "Path to browser profile")
	batchSize := flag.Int("batch", 10, "Number of dates per batch")
	execPath := flag.String("exec", "", "Browser executable (auto-detect if empty)")
	downloadDir := flag.String("download", defaultDownload, "Directory to save downloads")
	fromDate := flag.String("from", "", "Start date for filtering (format: YYYY-MM-DD)")
	toDate := flag.String("to", "", "End date for filtering (format: YYYY-MM-DD)")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("yandex-disk-photo-exporter version %s\n", appVersion)
		os.Exit(0)
	}

	// Auto-detect browser if not specified
	browserExec := *execPath
	if browserExec == "" {
		browserExec = browser.DetectBrowser()
		if browserExec == "" {
			log.Fatal("Error: Could not find Chrome/Chromium. Please install Chrome or specify path with -exec flag")
		}
		log.Printf("✓ Auto-detected browser: %s", browserExec)
	}

	// Expand ~ in download path
	downloadPath := *downloadDir
	if strings.HasPrefix(downloadPath, "~/") {
		downloadPath = filepath.Join(homeDir, downloadPath[2:])
	}

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		log.Fatalf("Error creating download directory: %v", err)
	}

	// Parse date range filter
	dateRange, err := datefilter.NewDateRange(*fromDate, *toDate)
	if err != nil {
		log.Fatalf("Error parsing date range: %v", err)
	}

	log.Println("=== Yandex Photo Downloader ===")
	log.Printf("Executable: %s", browserExec)
	log.Printf("Profile: %s", *profile)
	log.Printf("Download: %s", downloadPath)
	log.Printf("Batch: %d dates at a time", *batchSize)
	if dateRange.Enabled {
		log.Printf("Date range: %s", dateRange)
	}

	if err := run(*profile, *batchSize, browserExec, downloadPath, dateRange); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(profile string, batchSize int, execPath string, downloadDir string, dateRange *datefilter.DateRange) error {
	// Initialize stats for final report
	stats := report.New()
	defer stats.Print()

	// Initialize browser
	cfg := browser.DefaultConfig()
	cfg.ExecPath = execPath
	cfg.ProfilePath = profile
	cfg.DownloadDir = downloadDir

	browserCtx, err := browser.New(cfg)
	if err != nil {
		return err
	}
	defer browserCtx.Close()

	ctx := browserCtx.Ctx

	// 1. Open page
	log.Println("Opening Yandex Disk Photos...")
	if err := browser.Navigate(ctx, yandexPhotosURL); err != nil {
		return err
	}

	// Configure download directory
	if err := browser.ConfigureDownloads(ctx, downloadDir); err != nil {
		log.Printf("⚠️ Warning: could not configure download directory: %v", err)
	}

	// 2. Check login status
	isLoggedIn, err := auth.CheckLoginStatus(ctx)
	if err != nil {
		log.Printf("Warning: could not check login status: %v", err)
	}

	if !isLoggedIn {
		if err := auth.WaitForLogin(ctx); err != nil {
			return err
		}

		// Navigate to photos after successful login
		if err := browser.Navigate(ctx, yandexPhotosURL); err != nil {
			log.Printf("Warning: could not navigate after login: %v", err)
		}
	}

	log.Println("✓ User is logged in")

	// 3. Apply filter to show only photos from unlimited storage
	log.Println("Applying filter for unlimited storage photos...")
	if err := navigation.FilterByUnlimitedStorage(ctx); err != nil {
		log.Printf("⚠️ Warning: could not apply filter: %v", err)
		log.Println("Continuing without filter - all photos will be processed")
	}

	// Wait for page to update after filter
	time.Sleep(2 * time.Second)

	// 4. Main loop - process one date at a time
	emptyRounds := 0
	consecutiveErrors := 0
	const maxConsecutiveErrors = 3
	var currentDateInfo string // Track current date for error reporting

	for {
		// Check if browser/context is still valid
		if browser.IsContextCanceled(ctx) {
			log.Println("\n⚠️ Browser was closed. Exiting gracefully...")
			break
		}

		log.Printf("\n--- Processing date %d ---", stats.DatesProcessed+1)

		// Check for pending selection and clear it
		selection.ClearPendingSelection(ctx)

		// Select the FIRST visible date (always the top one)
		dateInfo, err := selection.SelectFirstVisibleDate(ctx)
		if err != nil {
			// Check if this is a fatal error (browser closed)
			if browser.IsBrowserClosed(err) {
				log.Println("\n⚠️ Browser was closed. Exiting gracefully...")
				break
			}
			log.Printf("Error selecting: %v", err)
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				log.Printf("⚠️ Too many consecutive errors (%d). Browser may be unresponsive.", consecutiveErrors)
				// Double-check if context is still valid
				if browser.IsContextCanceled(ctx) {
					log.Println("Browser context is no longer valid. Exiting...")
					break
				}
			}
			time.Sleep(1 * time.Second)
			continue
		}
		consecutiveErrors = 0 // Reset on success

		if dateInfo == nil {
			log.Println("No date found, scrolling...")
			if err := navigation.ScrollDown(ctx); err != nil {
				if browser.IsBrowserClosed(err) {
					log.Println("\n⚠️ Browser was closed. Exiting gracefully...")
					break
				}
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
		currentDateInfo = dateInfo.Text
		log.Println("✓ Date found: " + dateInfo.Text)

		// Check if date is within the specified range
		if dateRange.Enabled {
			inRange, err := dateRange.IsInRange(dateInfo.Text)
			if err != nil {
				log.Printf("⚠️ Could not parse date '%s': %v", dateInfo.Text, err)
				// Continue processing anyway if date can't be parsed
			} else if !inRange {
				// Check if we're past the range (dates are in reverse chronological order)
				if dateRange.IsBeforeRange(dateInfo.Text) {
					log.Printf("📅 Date '%s' is before the specified range. Stopping.", dateInfo.Text)
					// Deselect before stopping
					selection.Deselect(ctx)
					break
				}
				// Date is after range, skip it and scroll
				log.Printf("📅 Date '%s' is after the specified range. Skipping...", dateInfo.Text)
				stats.IncrementSkippedDates()
				selection.Deselect(ctx)
				time.Sleep(500 * time.Millisecond)
				if err := navigation.ScrollToPosition(ctx, dateInfo.YPosition); err != nil {
					log.Printf("Warning: scroll failed: %v", err)
				}
				time.Sleep(1 * time.Second)
				continue
			}
			log.Printf("✓ Date '%s' is within range", dateInfo.Text)
		}

		log.Println("✓ Date selected: " + dateInfo.Text)

		// Click Download
		time.Sleep(1500 * time.Millisecond)
		if err := download.ClickDownloadButton(ctx); err != nil {
			if browser.IsBrowserClosed(err) {
				log.Println("\n⚠️ Browser was closed. Exiting gracefully...")
				break
			}
			log.Printf("Download error: %v", err)
			stats.IncrementDownloadsFailed()
			stats.AddError(currentDateInfo, fmt.Sprintf("Download failed: %v", err))
		} else {
			log.Println("✓ Download started")
			stats.IncrementDownloadsStarted()
		}

		// Wait for download to start
		time.Sleep(4 * time.Second)

		// Deselect
		for retry := 0; retry < 3; retry++ {
			if err := selection.Deselect(ctx); err != nil {
				if browser.IsBrowserClosed(err) {
					log.Println("\n⚠️ Browser was closed. Exiting gracefully...")
					break
				}
				log.Printf("Error deselecting (attempt %d): %v", retry+1, err)
			}
			time.Sleep(1 * time.Second)

			if !selection.HasActiveSelection(ctx) {
				break
			}
			log.Printf("⚠️ Selection still active, trying again...")
		}

		// Check again if browser is still open before continuing
		if browser.IsContextCanceled(ctx) {
			log.Println("\n⚠️ Browser was closed. Exiting gracefully...")
			break
		}
		log.Println("✓ Deselected")

		// IMPORTANT: Scroll to move processed date off screen
		if err := navigation.ScrollToPosition(ctx, dateInfo.YPosition); err != nil {
			if browser.IsBrowserClosed(err) {
				log.Println("\n⚠️ Browser was closed. Exiting gracefully...")
				break
			}
			log.Printf("Warning: scroll to position failed: %v", err)
		}
		time.Sleep(1 * time.Second)

		stats.IncrementDatesProcessed()
	}

	log.Println("\nProcessing complete. Browser remains open. Press Ctrl+C to exit.")

	select {}
}
