package proxyruntime

import (
	"protosvpn.local/client/internal/client"
	"time"
)

type trafficSampler struct {
	at               time.Time
	upload, download uint64
	valid            bool
}

func (s *trafficSampler) sample(base *client.ConnectionMetrics, upload, download uint64, at time.Time) *client.ConnectionMetrics {
	m := client.CloneConnectionMetrics(base)
	m.SampledAt = at.UTC().Format(time.RFC3339Nano)
	m.UploadBytes = &upload
	m.DownloadBytes = &download
	if elapsed := at.Sub(s.at).Seconds(); s.valid && elapsed > 0 && upload >= s.upload && download >= s.download {
		up := float64(upload-s.upload) / elapsed
		down := float64(download-s.download) / elapsed
		m.UploadBps = &up
		m.DownloadBps = &down
	}
	s.at, s.upload, s.download, s.valid = at, upload, download, true
	return m
}
