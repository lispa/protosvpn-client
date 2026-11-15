package client

// ConnectionMetrics contains only actual per-session measurements, never profile secrets.
// Nil counters/rates mean unavailable, not a measured zero.
type ConnectionMetrics struct {
	StartedAt     string   `json:"startedAt"`
	SampledAt     string   `json:"sampledAt,omitempty"`
	LatencyMs     *int64   `json:"latencyMs,omitempty"`
	UploadBytes   *uint64  `json:"uploadBytes,omitempty"`
	DownloadBytes *uint64  `json:"downloadBytes,omitempty"`
	UploadBps     *float64 `json:"uploadBps,omitempty"`
	DownloadBps   *float64 `json:"downloadBps,omitempty"`
}

func CloneConnectionMetrics(m *ConnectionMetrics) *ConnectionMetrics {
	if m == nil {
		return nil
	}
	n := *m
	if m.LatencyMs != nil {
		v := *m.LatencyMs
		n.LatencyMs = &v
	}
	if m.UploadBytes != nil {
		v := *m.UploadBytes
		n.UploadBytes = &v
	}
	if m.DownloadBytes != nil {
		v := *m.DownloadBytes
		n.DownloadBytes = &v
	}
	if m.UploadBps != nil {
		v := *m.UploadBps
		n.UploadBps = &v
	}
	if m.DownloadBps != nil {
		v := *m.DownloadBps
		n.DownloadBps = &v
	}
	return &n
}
