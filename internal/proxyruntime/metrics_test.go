package proxyruntime

import (
	"protosvpn.local/client/internal/client"
	"testing"
	"time"
)

func TestTrafficSamplerUsesRealDeltasAndNeverInventsFirstRate(t *testing.T) {
	base := &client.ConnectionMetrics{StartedAt: "2026-01-01T00:00:00Z"}
	var s trafficSampler
	at := time.Unix(100, 0)
	first := s.sample(base, 1000, 2000, at)
	if first.UploadBps != nil || first.DownloadBps != nil || *first.UploadBytes != 1000 {
		t.Fatal("first sample invented a rate", first)
	}
	next := s.sample(base, 2000, 4000, at.Add(2*time.Second))
	if next.UploadBps == nil || *next.UploadBps != 500 || *next.DownloadBps != 1000 {
		t.Fatal("incorrect measured delta", next)
	}
	reset := s.sample(base, 1, 2, at.Add(4*time.Second))
	if reset.UploadBps != nil || reset.DownloadBps != nil {
		t.Fatal("counter reset caused fictitious rate")
	}
	if base.UploadBytes != nil {
		t.Fatal("sampler modified base snapshot")
	}
}
