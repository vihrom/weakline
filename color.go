//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
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

// isSGR reports whether val is a non-empty list of decimal numbers separated by ';'.
func isSGR(val string) bool {
	if val == "" {
		return false
	}
	for i := 0; i < len(val); i++ {
		char := val[i]
		if (char < '0' || char > '9') && char != ';' {
			return false
		}
	}
	return true
}

// parseHex converts "#RRGGBB", "RRGGBB" or the shorthand "#RGB" into a True Color ANSI sequence.
// The shorthand needs the '#', otherwise "196" would be ambiguous with a 256-palette index.
func parseHex(val string) (string, bool) {
	hex := val
	if strings.HasPrefix(val, "#") {
		hex = val[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
	}
	if len(hex) != 6 {
		return "", false
	}

	red, errRed := strconv.ParseUint(hex[0:2], 16, 8)
	green, errGreen := strconv.ParseUint(hex[2:4], 16, 8)
	blue, errBlue := strconv.ParseUint(hex[4:6], 16, 8)
	if errRed != nil || errGreen != nil || errBlue != nil {
		return "", false
	}

	res := []byte("38;2;")
	res = strconv.AppendUint(res, red, 10)
	res = append(res, ';')
	res = strconv.AppendUint(res, green, 10)
	res = append(res, ';')
	res = strconv.AppendUint(res, blue, 10)
	return string(res), true
}

// parse256 converts a color ID (0-255) into a 256-color ANSI sequence.
func parse256(val string) (string, bool) {
	if num, err := strconv.Atoi(val); err == nil && num >= 0 && num <= 255 {
		return "38;5;" + val, true
	}
	return "", false
}

// Map evaluates configuration string values and normalizes them into standard ANSI color definitions.
// Unknown values yield "" (a neutral sequence) so a typo can never inject garbage into the terminal.
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

	if isSGR(val) {
		return val
	}

	return ""
}
