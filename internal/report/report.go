// Package report provides final execution report functionality.
package report

import (
	"fmt"
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
	SkippedDates     int // Dates skipped (out of range)
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

// Finish marks the end time of the execution.
func (s *Stats) Finish() {
	s.EndTime = time.Now()
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

// Print outputs the final report to the console.
func (s *Stats) Print() {
	s.Finish()

	width := 54
	line := strings.Repeat("═", width-2)
	
	fmt.Println()
	fmt.Printf("╔%s╗\n", line)
	fmt.Printf("║%s║\n", centerText("📊 FINAL REPORT", width-2))
	fmt.Printf("╠%s╣\n", line)
	
	// Duration
	fmt.Printf("║  ⏱️  Duration:         %-28s║\n", formatDuration(s.Duration()))
	
	// Dates processed
	fmt.Printf("║  📅 Dates processed:   %-28d║\n", s.DatesProcessed)
	
	// Downloads
	downloadStr := fmt.Sprintf("%d started", s.DownloadsStarted)
	if s.DownloadsFailed > 0 {
		downloadStr += fmt.Sprintf(", %d failed", s.DownloadsFailed)
	}
	fmt.Printf("║  ⬇️  Downloads:         %-28s║\n", downloadStr)
	
	// Skipped dates (if any)
	if s.SkippedDates > 0 {
		fmt.Printf("║  ⏭️  Skipped:           %-28s║\n", fmt.Sprintf("%d (out of date range)", s.SkippedDates))
	}
	
	// Errors section
	if len(s.Errors) > 0 {
		fmt.Printf("╠%s╣\n", line)
		fmt.Printf("║  ❌ Errors (%d):%-36s║\n", len(s.Errors), "")
		
		// Show up to 5 errors
		maxErrors := 5
		for i, err := range s.Errors {
			if i >= maxErrors {
				remaining := len(s.Errors) - maxErrors
				fmt.Printf("║     ... and %d more errors%-24s║\n", remaining, "")
				break
			}
			errText := fmt.Sprintf("- %s", err.Message)
			if err.DateInfo != "" {
				errText += fmt.Sprintf(" (%s)", err.DateInfo)
			}
			// Truncate if too long
			if len(errText) > width-8 {
				errText = errText[:width-11] + "..."
			}
			fmt.Printf("║     %-48s║\n", errText)
		}
	} else {
		fmt.Printf("╠%s╣\n", line)
		fmt.Printf("║  ✅ No errors occurred%-30s║\n", "")
	}
	
	fmt.Printf("╚%s╝\n", line)
	fmt.Println()
}

// centerText centers a string within a given width.
func centerText(s string, width int) string {
	// Account for emoji width (some take 2 characters visually)
	visualLen := len(s)
	padding := (width - visualLen) / 2
	if padding < 0 {
		padding = 0
	}
	return fmt.Sprintf("%*s%s%*s", padding, "", s, width-padding-visualLen, "")
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
