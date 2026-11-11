//go:build !windows

package client

import (
	"os"
	"strings"
)

// Only return supported language codes, never the raw environment value.
func systemLocale() string {
	for _, key := range []string{"LC_ALL", "LANGUAGE", "LC_MESSAGES", "LANG"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if value == "" {
			continue
		}
		value = strings.SplitN(value, ":", 2)[0]
		if value == "ru" || strings.HasPrefix(value, "ru_") || strings.HasPrefix(value, "ru-") || strings.HasPrefix(value, "ru.") || strings.HasPrefix(value, "ru@") {
			return "ru"
		}
		return "en"
	}
	return "en"
}
