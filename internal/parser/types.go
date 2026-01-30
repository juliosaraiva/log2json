package parser

import (
	"strconv"
	"strings"
)

// inferType attempts to convert a string to its most appropriate type.
// Returns int64 for integers, float64 for decimals, bool for true/false,
// or the original string if no conversion applies.
func inferType(s string) any {
	// Try integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Try boolean
	lower := strings.ToLower(s)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}

	// Return as string
	return s
}
