package util

import "strings"

// StringToList parse the string value and return an array of string using either comma or space separator
func StringToList(s string) []string {
	if s == "" {
		return nil
	}

	var parts []string
	if strings.Contains(s, ",") {
		parts = strings.Split(s, ",")
	} else {
		parts = strings.Fields(s)
	}

	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return parts
}
