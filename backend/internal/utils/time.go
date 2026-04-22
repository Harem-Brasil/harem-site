package utils

import "time"

// FormatRFC3339UTC retorna timestamp em UTC ou string vazia para zero time.
func FormatRFC3339UTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
