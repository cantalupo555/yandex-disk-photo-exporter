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
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/download"
	"github.com/cantalupo555/yandex-disk-photo-exporter/internal/navigation"
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

	log.Println("=== Yandex Photo Downloader ===")
	log.Printf("Executable: %s", browserExec)
	log.Printf("Profile: %s", *profile)
	log.Printf("Download: %s", downloadPath)
	log.Printf("Batch: %d dates at a time", *batchSize)

	if err := run(*profile, *batchSize, browserExec, downloadPath); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(profile string, batchSize int, execPath string, downloadDir string) error {
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
	totalProcessed := 0
	emptyRounds := 0

	for {
		log.Printf("\n--- Processing date %d ---", totalProcessed+1)

		// Check for pending selection and clear it
		selection.ClearPendingSelection(ctx)

		// Select the FIRST visible date (always the top one)
		dateInfo, err := selection.SelectFirstVisibleDate(ctx)
		if err != nil {
			log.Printf("Error selecting: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if dateInfo == nil {
			log.Println("No date found, scrolling...")
			if err := navigation.ScrollDown(ctx); err != nil {
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
		log.Println("✓ Date selected: " + dateInfo.Text)

		// Click Download
		time.Sleep(1500 * time.Millisecond)
		if err := download.ClickDownloadButton(ctx); err != nil {
			log.Printf("Download error: %v", err)
		} else {
			log.Println("✓ Download started")
		}

		// Wait for download to start
		time.Sleep(4 * time.Second)

		// Deselect
		for retry := 0; retry < 3; retry++ {
			if err := selection.Deselect(ctx); err != nil {
				log.Printf("Error deselecting (attempt %d): %v", retry+1, err)
			}
			time.Sleep(1 * time.Second)

			if !selection.HasActiveSelection(ctx) {
				break
			}
			log.Printf("⚠️ Selection still active, trying again...")
		}
		log.Println("✓ Deselected")

		// IMPORTANT: Scroll to move processed date off screen
		if err := navigation.ScrollToPosition(ctx, dateInfo.YPosition); err != nil {
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
