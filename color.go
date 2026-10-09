package main

import (
	"fmt"
	"strconv"
	"strings"
)

// namedColors maps color identifiers to basic, bright, background, and styled ANSI escape codes.
var namedColors = map[string]string{
	// Reset & Styles
	"reset":     "0",
	"bold":      "1",
	"dim":       "2",
	"italic":    "3",
	"underline": "4",
	"reverse":   "7",

	// Standard Foreground (30-37)
	"black":   "30",
	"red":     "31",
	"green":   "32",
	"yellow":  "33",
	"blue":    "34",
	"magenta": "35",
	"cyan":    "36",
	"white":   "37",

	// Bold Standard Foreground
	"bold_black":   "30;1",
	"bold_red":     "31;1",
	"bold_green":   "32;1",
	"bold_yellow":  "33;1",
	"bold_blue":    "34;1",
	"bold_magenta": "35;1",
	"bold_cyan":    "36;1",
	"bold_white":   "37;1",

	// Bright High-Intensity Foreground (90-97)
	"gray":           "90",
	"bright_black":   "90",
	"bright_red":     "91",
	"bright_green":   "92",
	"bright_yellow":  "93",
	"bright_blue":    "94",
	"bright_magenta": "95",
	"bright_cyan":    "96",
	"bright_white":   "97",

	// Bold Bright Foreground
	"bold_gray":           "90;1",
	"bold_bright_black":   "90;1",
	"bold_bright_red":     "91;1",
	"bold_bright_green":   "92;1",
	"bold_bright_yellow":  "93;1",
	"bold_bright_blue":    "94;1",
	"bold_bright_magenta": "95;1",
	"bold_bright_cyan":    "96;1",
	"bold_bright_white":   "97;1",

	// Standard Backgrounds (40-47)
	"bg_black":   "40",
	"bg_red":     "41",
	"bg_green":   "42",
	"bg_yellow":  "43",
	"bg_blue":    "44",
	"bg_magenta": "45",
	"bg_cyan":    "46",
	"bg_white":   "47",

	// Bright Backgrounds (100-107)
	"bg_bright_black":   "100",
	"bg_bright_red":     "101",
	"bg_bright_green":   "102",
	"bg_bright_yellow":  "103",
	"bg_bright_blue":    "104",
	"bg_bright_magenta": "105",
	"bg_bright_cyan":    "106",
	"bg_bright_white":   "107",
}

// isHex checks if a string consists entirely of valid hexadecimal characters.
func isHex(s string) bool {
	_, err := strconv.ParseUint(s, 16, 64)
	return err == nil
}

// parseHex converts a HEX string (#RRGGBB or RRGGBB) into a True Color ANSI sequence.
func parseHex(val string) (string, bool) {
	hex := strings.TrimPrefix(val, "#")
	if len(hex) != 6 || !isHex(hex) {
		return "", false
	}

	var r, g, b uint8
	if n, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); err == nil && n == 3 {
		res := []byte("38;2;")
		res = strconv.AppendUint(res, uint64(r), 10)
		res = append(res, ';')
		res = strconv.AppendUint(res, uint64(g), 10)
		res = append(res, ';')
		res = strconv.AppendUint(res, uint64(b), 10)
		return string(res), true
	}

	return "", false
}

// parse256 converts a color ID (0-255) into a 256-color ANSI sequence.
func parse256(val string) (string, bool) {
	if num, err := strconv.Atoi(val); err == nil && num >= 0 && num <= 255 {
		return "38;5;" + val, true
	}
	return "", false
}

// Map evaluates configuration string values and normalizes them into standard ANSI color definitions.
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
