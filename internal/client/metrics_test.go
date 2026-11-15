package client

import (
	"context"
	"testing"
)

type metricsConnector struct {
	fakeConnector
	metrics *ConnectionMetrics
}

func (f *metricsConnector) Metrics() *ConnectionMetrics { return f.metrics }
func TestConnectionPublishesOnlyActiveMetricsAndCopiesNumbers(t *testing.T) {
	up := uint64(1234)
	latency := int64(12)
	f := &metricsConnector{metrics: &ConnectionMetrics{StartedAt: "2026-01-01T00:00:00Z", UploadBytes: &up, LatencyMs: &latency}}
	s := NewWithConnector(f)
	p, _ := s.Import("vless", testURI, "")
	if s.State().Connection.Metrics != nil {
		t.Fatal("disconnected metrics leaked")
	}
	if err := s.Connect(context.Background(), p.ID); err != nil {
		t.Fatal(err)
	}
	m := s.State().Connection.Metrics
	if m == nil || m.UploadBytes == nil || *m.UploadBytes != 1234 || m.LatencyMs == nil || *m.LatencyMs != 12 {
		t.Fatal("active metrics missing", m)
	}
	*m.UploadBytes = 999
	*m.LatencyMs = 99
	if *s.State().Connection.Metrics.UploadBytes != 1234 || *f.metrics.LatencyMs != 12 {
		t.Fatal("state snapshot aliases provider metrics")
	}
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if s.State().Connection.Metrics != nil {
		t.Fatal("old session metrics survived disconnect")
	}
}
