// Package browser provides Chrome/Chromedp initialization and configuration.
package browser

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DetectBrowser attempts to find a Chrome/Chromium executable on the system.
// Returns the path to the executable, or empty string if not found.
func DetectBrowser() string {
	var candidates []string

	switch runtime.GOOS {
	case "windows":
		candidates = getWindowsCandidates()
	case "darwin":
		candidates = getMacOSCandidates()
	default: // linux and others
		candidates = getLinuxCandidates()
	}

	// Check each candidate path
	for _, path := range candidates {
		if path == "" {
			continue
		}
		// Expand environment variables (for Windows %LOCALAPPDATA% etc.)
		expanded := os.ExpandEnv(path)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}

	// Fallback: try to find in PATH
	for _, name := range []string{"chrome", "chromium", "chromium-browser", "google-chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	return ""
}

// getWindowsCandidates returns common Chrome/Chromium paths on Windows.
// Priority: Chrome > Chromium > Edge > Vivaldi > Opera > Brave
func getWindowsCandidates() []string {
	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")

	return []string{
		// 1. Chrome (highest priority)
		filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(programFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
		// 2. Chromium
		filepath.Join(programFiles, "Chromium", "Application", "chrome.exe"),
		filepath.Join(programFilesX86, "Chromium", "Application", "chrome.exe"),
		filepath.Join(localAppData, "Chromium", "Application", "chrome.exe"),
		// 3. Edge
		filepath.Join(programFiles, "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(programFilesX86, "Microsoft", "Edge", "Application", "msedge.exe"),
		// 4. Vivaldi
		filepath.Join(localAppData, "Vivaldi", "Application", "vivaldi.exe"),
		filepath.Join(programFiles, "Vivaldi", "Application", "vivaldi.exe"),
		// 5. Opera
		filepath.Join(localAppData, "Programs", "Opera", "opera.exe"),
		filepath.Join(programFiles, "Opera", "opera.exe"),
		// 6. Brave (lowest priority)
		filepath.Join(programFiles, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
	}
}

// getMacOSCandidates returns common Chrome/Chromium paths on macOS.
// Priority: Chrome > Chromium > Edge > Vivaldi > Opera > Brave
func getMacOSCandidates() []string {
	return []string{
		// 1. Chrome (highest priority)
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		os.ExpandEnv("$HOME/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
		// 2. Chromium
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		os.ExpandEnv("$HOME/Applications/Chromium.app/Contents/MacOS/Chromium"),
		// 3. Edge
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		// 4. Vivaldi
		"/Applications/Vivaldi.app/Contents/MacOS/Vivaldi",
		// 5. Opera
		"/Applications/Opera.app/Contents/MacOS/Opera",
		// 6. Brave (lowest priority)
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	}
}

// getLinuxCandidates returns common Chrome/Chromium paths on Linux.
// Priority: Chrome > Chromium > Edge > Vivaldi > Opera > Brave
func getLinuxCandidates() []string {
	return []string{
		// 1. Chrome (highest priority)
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/var/lib/flatpak/exports/bin/com.google.Chrome",
		// 2. Chromium
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/snap/bin/chromium",
		"/var/lib/flatpak/exports/bin/org.chromium.Chromium",
		// 3. Edge
		"/usr/bin/microsoft-edge-stable",
		"/usr/bin/microsoft-edge",
		// 4. Vivaldi
		"/usr/bin/vivaldi",
		"/usr/bin/vivaldi-stable",
		// 5. Opera
		"/usr/bin/opera",
		// 6. Brave (lowest priority)
		"/usr/bin/brave-browser",
		"/opt/brave.com/brave/brave-browser",
	}
}

// DefaultProfilePath returns the default profile path for the current OS.
func DefaultProfilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "windows":
		// Dedicated profile to avoid conflicts with user's main browser
		return filepath.Join(homeDir, ".yandex-exporter-profile")
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "yandex-exporter-profile")
	default: // linux
		// Check if snap chromium profile exists
		snapPath := filepath.Join(homeDir, "snap", "chromium", "common", "chromium")
		if _, err := os.Stat(snapPath); err == nil {
			return snapPath
		}
		// Default to standard chromium config
		return filepath.Join(homeDir, ".config", "chromium")
	}
}
