package main

import (
	"fmt"
	"strconv"
	"strings"
)

// namedColors maps color identifiers to basic 3/4-bit ANSI codes.
var namedColors = map[string]string{
	"black":        "30",
	"red":          "31",
	"green":        "32",
	"yellow":       "33",
	"blue":         "34",
	"magenta":      "35",
	"cyan":         "36",
	"white":        "37",
	"gray":         "90",
	"bold_black":   "30;1",
	"bold_red":     "31;1",
	"bold_green":   "32;1",
	"bold_yellow":  "33;1",
	"bold_blue":    "34;1",
	"bold_magenta": "35;1",
	"bold_cyan":    "36;1",
	"bold_white":   "37;1",
	"bold_gray":    "90;1",
}

// isHex checks if a string consists entirely of valid hexadecimal characters.
func isHex(s string) bool {
	_, err := strconv.ParseUint(s, 16, 64)
	return err == nil
}

// parseHex converts a HEX string (#RRGGBB or RRGGBB) into a True Color ANSI sequence.
func parseHex(val string) (string, bool) {
	if !strings.HasPrefix(val, "#") && (len(val) != 6 || !isHex(val)) {
		return "", false
	}

	hex := strings.TrimPrefix(val, "#")
	var r, g, b uint8
	if n, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); err == nil && n == 3 {
		return fmt.Sprintf("38;2;%d;%d;%d", r, g, b), true
	}

	return "", false
}

// parse256 converts a color ID (0-255) into a 256-color ANSI sequence.
func parse256(val string) (string, bool) {
	if num, err := strconv.Atoi(val); err == nil && num >= 0 && num <= 255 {
		return fmt.Sprintf("38;5;%s", val), true
	}
	return "", false
}

// Map evaluates configuration string values and normalizes them into standard ANSI color definitions.
// It seamlessly supports True Color (HEX), 8-bit (256-color palette), named presets, or raw terminal codes.
func Map(val string) string {
	val = strings.TrimSpace(val)

	if ansi, ok := parseHex(val); ok {
		return ansi
	}

	if ansi, ok := parse256(val); ok {
		return ansi
	}

	if code, ok := namedColors[val]; ok {
		return code
	}

	return val
}
