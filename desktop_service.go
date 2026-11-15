package main

import (
	"protosvpn.local/client/internal/client"
	"protosvpn.local/client/internal/proxyruntime"
)

func newDesktopService(executable string) *client.Service {
	return client.NewPersistentWithConnector(proxyruntime.New(executable))
}
