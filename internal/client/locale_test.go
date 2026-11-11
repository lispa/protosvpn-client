//go:build !windows

package client

import (
	"encoding/json"
	"testing"
)

func TestStateReportsSupportedSystemLanguage(t *testing.T) {
	for _, tc := range []struct{ locale, want string }{{"ru_RU.UTF-8", "ru"}, {"ru-FI", "ru"}, {"en_US.UTF-8", "en"}, {"fi_FI.UTF-8", "en"}, {"C", "en"}} {
		t.Run(tc.locale, func(t *testing.T) {
			t.Setenv("LC_ALL", tc.locale)
			b, _ := json.Marshal(New().State())
			var state map[string]any
			json.Unmarshal(b, &state)
			if state["systemLocale"] != tc.want {
				t.Fatalf("systemLocale=%v, want %s", state["systemLocale"], tc.want)
			}
		})
	}
}
