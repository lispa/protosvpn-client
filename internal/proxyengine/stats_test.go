package proxyengine

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStatsOutputBounded(t *testing.T) {
	output := &statsOutput{}
	if _, err := io.Copy(output, io.LimitReader(strings.NewReader(strings.Repeat("s", maxStatsOutput*2)), maxStatsOutput*2)); err != nil {
		t.Fatal(err)
	}
	if len(output.Bytes()) != maxStatsOutput || !output.overflow {
		t.Fatal("subprocess output limit bypassed")
	}
}

func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "api" && os.Args[2] == "statsquery" {
		// An actual subprocess fixture verifies the production no-shell CLI path.
		if len(os.Args) != 8 || os.Args[3] != "--server=127.0.0.1:1" || os.Args[4] != "-pattern" || os.Args[5] != statsPattern || os.Args[6] != "-timeout" || os.Args[7] != "3" {
			os.Exit(2)
		}
		switch os.Getenv("PROXYENGINE_STATS_HELPER") {
		case "success":
			_, _ = io.WriteString(os.Stdout, `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":123},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":456}]}`)
		case "malformed":
			_, _ = io.WriteString(os.Stdout, `secret-uuid-server`)
		case "oversized":
			_, _ = io.WriteString(os.Stdout, strings.Repeat("s", maxStatsOutput*2))
		case "failure":
			_, _ = io.WriteString(os.Stderr, `secret-uuid-server`)
			os.Exit(1)
		case "hang":
			time.Sleep(30 * time.Second)
		default:
			os.Exit(2)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestStatsCLISanitization(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"success", "malformed", "oversized", "failure", "hang"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("PROXYENGINE_STATS_HELPER", mode)
			e := New(binary)
			e.ready = true
			e.statsEndpoint = "127.0.0.1:1"
			defer e.Close()
			ctx := context.Background()
			if mode == "hang" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			got, err := e.Stats(ctx)
			if mode == "success" {
				if err != nil || got != (Traffic{123, 456}) {
					t.Fatal("CLI invocation failed", err)
				}
				return
			}
			if err != ErrStatsUnavailable || got != (Traffic{}) || strings.Contains(err.Error(), "secret") {
				t.Fatal("unsafe/unavailable CLI result accepted")
			}
		})
	}
}

func TestStatsParser(t *testing.T) {
	for _, tc := range []struct {
		data string
		want Traffic
	}{
		{`{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":123},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":456}]}`, Traffic{123, 456}},
		{`{"stat":[{"name":"outbound>>>proxy>>>traffic>>>downlink"},{"name":"outbound>>>proxy>>>traffic>>>uplink"}]}`, Traffic{}},
	} {
		got, err := parseStats([]byte(tc.data))
		if err != nil || got != tc.want {
			t.Fatalf("valid stats rejected: %+v %v", got, err)
		}
	}
	valid := `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":123},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":456}]}`
	for _, data := range []string{
		``, `{}`, `{"stat":null}`, `{"stat":[]}`, `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink"}]}`,
		valid + `{}`, strings.Replace(valid, `"value":123`, `"value":-1`, 1),
		strings.Replace(valid, `"value":123`, `"value":9223372036854775808`, 1),
		strings.Replace(valid, `"value":123`, `"value":1.5`, 1),
		strings.Replace(valid, `"value":123`, `"value":null`, 1),
		strings.Replace(valid, `"value":123`, `"value":"123"`, 1),
		strings.Replace(valid, `"value":123`, `"value":123,"value":0`, 1),
		strings.Replace(valid, `"stat":`, `"secret":"secret-uuid-server","stat":`, 1),
		strings.Replace(valid, `>>>downlink`, `>>>uplink`, 1),
		strings.Replace(valid, `outbound>>>proxy`, `user>>>secret-uuid-server`, 1),
		strings.Repeat(" ", 4097) + valid,
	} {
		got, err := parseStats([]byte(data))
		if err != ErrStatsUnavailable || got != (Traffic{}) {
			t.Fatalf("unsafe stats accepted: %+v %v", got, err)
		}
	}
}

func TestStatsUnavailableBeforeStart(t *testing.T) {
	e := New("secret-uuid-server")
	if got, err := e.Stats(context.Background()); err != ErrStatsUnavailable || got != (Traffic{}) {
		t.Fatal("unstarted counters accepted")
	}
	e.Close()
	if _, err := e.Stats(context.Background()); err != ErrStatsUnavailable {
		t.Fatal("closed counters accepted")
	}
}

func TestStatsConfigIsIsolated(t *testing.T) {
	b, err := makeConfigWithStats(baseURI, 12345, 12346)
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	api := c["api"].(map[string]any)
	if api["listen"] != "127.0.0.1:12346" || len(api["services"].([]any)) != 1 || api["services"].([]any)[0] != "StatsService" {
		t.Fatal("API not isolated/minimal")
	}
	if len(c["inbounds"].([]any)) != 1 || c["routing"] != nil {
		t.Fatal("API leaked into proxy routing")
	}
	if c["outbounds"].([]any)[0].(map[string]any)["tag"] != "proxy" {
		t.Fatal("missing stable outbound tag")
	}
	policy := c["policy"].(map[string]any)
	system := policy["system"].(map[string]any)
	if len(policy) != 1 || len(system) != 2 || system["statsOutboundUplink"] != true || system["statsOutboundDownlink"] != true {
		t.Fatal("non-outbound stats enabled")
	}
	if _, ok := c["stats"]; !ok {
		t.Fatal("stats manager missing")
	}
	for _, port := range []int{0, -1, 65536, 12345} {
		if _, err := makeConfigWithStats(baseURI, 12345, port); err != ErrInvalid {
			t.Fatal("invalid/private port accepted")
		}
	}
}
