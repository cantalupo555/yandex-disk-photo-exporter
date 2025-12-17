// Package browser provides Chrome/Chromedp initialization and configuration.
package browser

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

// Config holds browser configuration options.
type Config struct {
	ExecPath    string
	ProfilePath string
	DownloadDir string
	WindowWidth int
	WindowHeight int
	Timeout     time.Duration
}

// DefaultConfig returns default browser configuration.
func DefaultConfig() Config {
	return Config{
		ExecPath:     "chromium",
		WindowWidth:  1920,
		WindowHeight: 1080,
		Timeout:      2 * time.Hour,
	}
}

// Context holds the browser contexts and cancel functions.
type Context struct {
	Ctx         context.Context
	AllocCancel context.CancelFunc
	CtxCancel   context.CancelFunc
}

// New creates a new browser context with the given configuration.
func New(cfg Config) (*Context, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(cfg.ExecPath),
		chromedp.UserDataDir(cfg.ProfilePath),
		chromedp.Flag("headless", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(cfg.WindowWidth, cfg.WindowHeight),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	ctx, timeoutCancel := context.WithTimeout(ctx, cfg.Timeout)

	// Wrap both cancels
	combinedCancel := func() {
		timeoutCancel()
		ctxCancel()
	}

	return &Context{
		Ctx:         ctx,
		AllocCancel: allocCancel,
		CtxCancel:   combinedCancel,
	}, nil
}

// Close closes all browser contexts.
func (c *Context) Close() {
	if c.CtxCancel != nil {
		c.CtxCancel()
	}
	if c.AllocCancel != nil {
		c.AllocCancel()
	}
}

// Navigate navigates to the given URL.
func Navigate(ctx context.Context, url string) error {
	return chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second),
	)
}

// ConfigureDownloads sets up the download directory for the browser.
func ConfigureDownloads(ctx context.Context, downloadDir string) error {
	if err := chromedp.Run(ctx,
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
			WithDownloadPath(downloadDir).
			WithEventsEnabled(true),
	); err != nil {
		return err
	}
	log.Printf("✓ Downloads will be saved to: %s", downloadDir)
	return nil
}

// GetCurrentURL returns the current page URL.
func GetCurrentURL(ctx context.Context) (string, error) {
	var url string
	if err := chromedp.Run(ctx, chromedp.Location(&url)); err != nil {
		return "", err
	}
	return url, nil
}
