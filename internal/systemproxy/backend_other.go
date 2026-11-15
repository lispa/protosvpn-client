//go:build !windows

package systemproxy

func newBackend() backend { return nil }
