// Package auth provides authentication verification for Yandex services.
package auth

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	// LoginCheckInterval is how often to check login status when waiting.
	LoginCheckInterval = 10 * time.Second
	// LoginTimeout is the maximum time to wait for user login.
	LoginTimeout = 5 * time.Minute
)

// CheckLoginStatus verifies if the user is logged into Yandex.
// Returns true if logged in, false if on login page.
func CheckLoginStatus(ctx context.Context) (bool, error) {
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

// WaitForLogin waits for the user to complete login within the timeout period.
// Returns nil if login is successful, error if timeout or check fails.
func WaitForLogin(ctx context.Context) error {
	log.Println("⚠️  User is NOT logged in!")
	log.Println("⚠️  Please log in to your Yandex account in the browser window.")
	log.Printf("Waiting for login (checking every %v, max %v)...", LoginCheckInterval, LoginTimeout)

	loginTimeout := time.After(LoginTimeout)
	loginCheck := time.NewTicker(LoginCheckInterval)
	defer loginCheck.Stop()

	for {
		select {
		case <-loginTimeout:
			return fmt.Errorf("login timeout: user did not log in within %v", LoginTimeout)
		case <-loginCheck.C:
			isLoggedIn, err := CheckLoginStatus(ctx)
			if err != nil {
				log.Printf("Warning: login check failed: %v", err)
				continue
			}
			if isLoggedIn {
				log.Println("✓ Login detected!")
				return nil
			}
			log.Println("Still waiting for login...")
		}
	}
}
