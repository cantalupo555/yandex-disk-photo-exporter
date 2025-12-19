// Package report provides final execution report functionality.
package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrorEntry represents a single error that occurred during execution.
type ErrorEntry struct {
	Timestamp time.Time
	DateInfo  string // The date being processed when error occurred
	Message   string
}

// Stats holds all statistics collected during execution.
type Stats struct {
	StartTime        time.Time
	EndTime          time.Time
	DatesProcessed   int
	DownloadsStarted int
	DownloadsFailed  int
	SkippedDates     int   // Dates skipped (out of range)
	TotalSize        int64 // Total size of downloaded files in bytes
	DownloadDir      string
	Errors           []ErrorEntry
}

// New creates a new Stats instance with StartTime set to now.
func New() *Stats {
	return &Stats{
		StartTime: time.Now(),
		Errors:    make([]ErrorEntry, 0),
	}
}

// AddError records an error that occurred during processing.
func (s *Stats) AddError(dateInfo, message string) {
	s.Errors = append(s.Errors, ErrorEntry{
		Timestamp: time.Now(),
		DateInfo:  dateInfo,
		Message:   message,
	})
}

// IncrementDownloadsStarted increments the successful downloads counter.
func (s *Stats) IncrementDownloadsStarted() {
	s.DownloadsStarted++
}

// IncrementDownloadsFailed increments the failed downloads counter.
func (s *Stats) IncrementDownloadsFailed() {
	s.DownloadsFailed++
}

// IncrementDatesProcessed increments the processed dates counter.
func (s *Stats) IncrementDatesProcessed() {
	s.DatesProcessed++
}

// IncrementSkippedDates increments the skipped dates counter.
func (s *Stats) IncrementSkippedDates() {
	s.SkippedDates++
}

// Finish marks the end time of the execution and calculates final stats.
func (s *Stats) Finish() {
	s.EndTime = time.Now()
	// Calculate total size of downloaded files
	if s.DownloadDir != "" {
		s.TotalSize = calculateDirSize(s.DownloadDir)
	}
}

// SetDownloadDir sets the download directory for size calculation.
func (s *Stats) SetDownloadDir(dir string) {
	s.DownloadDir = dir
}

// calculateDirSize returns the total size of all files in a directory.
func calculateDirSize(dir string) int64 {
	var size int64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// formatBytes formats bytes into human-readable format.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// Duration returns the total execution duration.
func (s *Stats) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// Print outputs the final report to the console with colors.
func (s *Stats) Print() {
	s.Finish()

	// Box width (internal content width, excluding borders)
	contentWidth := 52
	
	fmt.Println()
	printBoxTop(contentWidth)
	printBoxTitle("📊 FINAL REPORT", contentWidth)
	printBoxSeparator(contentWidth)
	
	// Duration
	printDataRow("⏱️ ", "Duration", formatDuration(s.Duration()), contentWidth, "")
	
	// Dates processed
	printDataRow("📅", "Dates processed", fmt.Sprintf("%d", s.DatesProcessed), contentWidth, "")
	
	// Downloads
	downloadValue := fmt.Sprintf("%d started", s.DownloadsStarted)
	downloadColor := colorGreen
	if s.DownloadsFailed > 0 {
		downloadValue += fmt.Sprintf(", %d failed", s.DownloadsFailed)
		downloadColor = colorYellow
	}
	printDataRow("⬇️ ", "Downloads", downloadValue, contentWidth, downloadColor)
	
	// Total size
	if s.TotalSize > 0 {
		printDataRow("💾", "Total size", formatBytes(s.TotalSize), contentWidth, "")
	}
	
	// Skipped dates (if any)
	if s.SkippedDates > 0 {
		skippedValue := fmt.Sprintf("%d (out of date range)", s.SkippedDates)
		printDataRow("⏭️ ", "Skipped", skippedValue, contentWidth, colorYellow)
	}
	
	// Errors section
	printBoxSeparator(contentWidth)
	if len(s.Errors) > 0 {
		errTitle := fmt.Sprintf("Errors (%d):", len(s.Errors))
		printDataRow("❌", errTitle, "", contentWidth, colorRed)
		
		// Show up to 5 errors
		maxErrors := 5
		for i, err := range s.Errors {
			if i >= maxErrors {
				remaining := len(s.Errors) - maxErrors
				printErrorLine(fmt.Sprintf("... and %d more errors", remaining), contentWidth)
				break
			}
			errText := fmt.Sprintf("- %s", err.Message)
			if err.DateInfo != "" {
				errText += fmt.Sprintf(" (%s)", err.DateInfo)
			}
			printErrorLine(errText, contentWidth)
		}
	} else {
		printDataRow("✅", "No errors occurred", "", contentWidth, colorGreen)
	}
	
	printBoxBottom(contentWidth)
	fmt.Println()
}

// printBoxTop prints the top border.
func printBoxTop(width int) {
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("=", width), colorReset)
}

// printBoxBottom prints the bottom border.
func printBoxBottom(width int) {
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("=", width), colorReset)
}

// printBoxSeparator prints a horizontal separator line.
func printBoxSeparator(width int) {
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("-", width), colorReset)
}

// printBoxTitle prints a centered title.
func printBoxTitle(title string, width int) {
	visLen := measureString(title)
	padding := (width - visLen) / 2
	if padding < 0 { padding = 0 }
	
	fmt.Printf("%s%s%s%s%s\n", 
		strings.Repeat(" ", padding),
		colorBold, title, colorReset,
		colorCyan) // Restore color for next lines if needed, though mostly reset
}

// printDataRow prints a data row with emoji, label, and value.
func printDataRow(emoji, label, value string, width int, valueColor string) {
	// Layout: "  [emoji] [label] [SPACER] [value]"
	// IDent: 2 spaces
	indent := "  "
	
	colGap := "   " // Space between label and value
	
	labelFixedVisWidth := 22
	
	// Prepare Label
	fullLabel := label
	if emoji != "" {
		fullLabel = emoji + "  " + label // Extra space after emoji for aesthetics
	}
	
	labelVis := measureString(fullLabel)
	labelPadding := labelFixedVisWidth - labelVis
	if labelPadding < 0 { labelPadding = 0 }
	
	labelField := fullLabel + strings.Repeat(" ", labelPadding)

	valueField := value
	if valueColor != "" {
		valueField = valueColor + value + colorReset
	}

	fmt.Printf("%s%s%s%s%s%s\n", 
		colorCyan, // Base color (though mostly reset inside)
		indent,
		colorReset + labelField,
		colGap,
		valueField,
		colorReset)
}

// printErrorLine prints an error detail line.
func printErrorLine(text string, width int) {
	// Layout: "      [text]"
	indent := "      " // Indent to align with text start of data rows
	
	fmt.Printf("%s%s%s%s\n",
		colorCyan, 
		indent,
		colorRed + text + colorReset,
		colorReset)
}

// measureString returns visual length of string without ANSI codes
func measureString(s string) int {
	return visualLength(stripAnsiCodes(s))
}

// visualLength calculates visual width of string handling emojis
func visualLength(s string) int {
	width := 0
	for _, r := range s {
		// Variation Selector-16 (VS16) to force emoji style: \uFE0F
		// Some chars are zero width
		if r == '\ufe0f' {
			continue
		}
		
		// Simple heuristic for East Asian Width / Emojis
		// Most emojis and CJK characters are 2 width
		// ASCII and simple unicode are 1
		if r > 256 {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// stripAnsiCodes removes ANSI escape codes from a string.
func stripAnsiCodes(s string) string {
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}

// Summary returns a brief one-line summary of the stats.
func (s *Stats) Summary() string {
	return fmt.Sprintf(
		"%d dates processed, %d downloads (%d failed), %d skipped, %d errors in %s",
		s.DatesProcessed,
		s.DownloadsStarted,
		s.DownloadsFailed,
		s.SkippedDates,
		len(s.Errors),
		formatDuration(s.Duration()),
	)
}
