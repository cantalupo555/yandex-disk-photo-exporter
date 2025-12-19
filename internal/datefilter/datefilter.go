// Package datefilter provides date range filtering functionality for photo selection.
package datefilter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateRange represents a date range filter.
type DateRange struct {
	From    time.Time
	To      time.Time
	Enabled bool
}

// NewDateRange creates a new DateRange from string dates.
// Date format: YYYY-MM-DD (e.g., "2023-01-01")
// Pass empty strings to disable filtering.
func NewDateRange(from, to string) (*DateRange, error) {
	dr := &DateRange{}

	if from == "" && to == "" {
		dr.Enabled = false
		return dr, nil
	}

	dr.Enabled = true

	if from != "" {
		fromDate, err := time.Parse("2006-01-02", from)
		if err != nil {
			return nil, fmt.Errorf("invalid 'from' date format (use YYYY-MM-DD): %w", err)
		}
		dr.From = fromDate
	}

	if to != "" {
		toDate, err := time.Parse("2006-01-02", to)
		if err != nil {
			return nil, fmt.Errorf("invalid 'to' date format (use YYYY-MM-DD): %w", err)
		}
		dr.To = toDate
	}

	// If only one date is specified, use sensible defaults
	if from != "" && to == "" {
		// From date to today
		dr.To = time.Now()
	}
	if to != "" && from == "" {
		// From beginning of time to specified date
		dr.From = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	// Validate range
	if dr.From.After(dr.To) {
		return nil, fmt.Errorf("'from' date (%s) is after 'to' date (%s)", from, to)
	}

	return dr, nil
}

// monthMap maps English month names to month numbers.
var monthMap = map[string]time.Month{
	"january":   time.January,
	"february":  time.February,
	"march":     time.March,
	"april":     time.April,
	"may":       time.May,
	"june":      time.June,
	"july":      time.July,
	"august":    time.August,
	"september": time.September,
	"october":   time.October,
	"november":  time.November,
	"december":  time.December,
}

// datePattern matches "12 January" or "12 January 2023" format.
var datePattern = regexp.MustCompile(`^(\d{1,2})\s+([A-Za-z]+)(?:\s+(\d{4}))?$`)

// ParseYandexDate parses a date string from Yandex Disk format.
// Formats: "12 January" (assumes current year) or "12 January 2023"
func ParseYandexDate(dateText string) (time.Time, error) {
	dateText = strings.TrimSpace(dateText)
	matches := datePattern.FindStringSubmatch(dateText)

	if matches == nil {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateText)
	}

	day, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", matches[1])
	}

	monthName := strings.ToLower(matches[2])
	month, ok := monthMap[monthName]
	if !ok {
		return time.Time{}, fmt.Errorf("invalid month: %s", matches[2])
	}

	year := time.Now().Year()
	if matches[3] != "" {
		year, err = strconv.Atoi(matches[3])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid year: %s", matches[3])
		}
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

// IsInRange checks if a date text (e.g., "12 January") is within the date range.
// Returns true if filtering is disabled or the date is within range.
// Returns an error if the date cannot be parsed.
func (dr *DateRange) IsInRange(dateText string) (bool, error) {
	if !dr.Enabled {
		return true, nil
	}

	parsedDate, err := ParseYandexDate(dateText)
	if err != nil {
		return false, err
	}

	// Check if date is within range (inclusive)
	if parsedDate.Before(dr.From) {
		return false, nil
	}
	if parsedDate.After(dr.To) {
		return false, nil
	}

	return true, nil
}

// IsBeforeRange checks if a date is before the range start.
// This is useful to know when to stop processing (dates are chronological).
func (dr *DateRange) IsBeforeRange(dateText string) bool {
	if !dr.Enabled {
		return false
	}

	parsedDate, err := ParseYandexDate(dateText)
	if err != nil {
		return false
	}

	return parsedDate.Before(dr.From)
}

// IsAfterRange checks if a date is after the range end.
// This is useful to know when to start processing.
func (dr *DateRange) IsAfterRange(dateText string) bool {
	if !dr.Enabled {
		return false
	}

	parsedDate, err := ParseYandexDate(dateText)
	if err != nil {
		return false
	}

	return parsedDate.After(dr.To)
}

// String returns a human-readable representation of the date range.
func (dr *DateRange) String() string {
	if !dr.Enabled {
		return "all dates"
	}
	return fmt.Sprintf("%s to %s", dr.From.Format("2006-01-02"), dr.To.Format("2006-01-02"))
}
