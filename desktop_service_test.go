package main

import (
	"testing"
)

func TestDesktopServiceRestoresProfilesAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	a := newDesktopService("unused")
	p, err := a.Import("vless", "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls#Saved", "")
	if err != nil {
		t.Fatal(err)
	}
	b := newDesktopService("unused")
	state := b.State()
	if len(state.Profiles) != 1 || state.Profiles[0].ID != p.ID || state.StorageMode != "protected" {
		t.Fatal("desktop lost profile after reopen", state.StorageMode, len(state.Profiles))
	}
	if state.Connection.Status != "disconnected" || state.Connection.Compatibility {
		t.Fatal("restart restored connection or legacy consent")
	}
}
