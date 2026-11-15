//go:build desktop && !native_smoke

package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"io/fs"
	"os"
	"protosvpn.local/client/internal/client"
	"protosvpn.local/client/internal/proxyruntime"
)

//go:embed all:frontend/dist
var embeddedFrontend embed.FS

func main() {
	if proxyruntime.MaybeRun() {
		return
	}
	assets, err := fs.Sub(embeddedFrontend, "frontend/dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Не удалось загрузить интерфейс ProtosVPN.")
		os.Exit(1)
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to locate application.")
		return
	}
	service := newDesktopService(executable)
	defer service.Close()
	options := newDesktopOptions(assets, client.NewHandler(service))
	options.OnBeforeClose = func(context.Context) bool { return service.Close() != nil }
	options.OnShutdown = func(context.Context) { _ = service.Close() }
	if err = wails.Run(options); err != nil {
		fmt.Fprintln(os.Stderr, "Не удалось запустить ProtosVPN. Проверьте наличие WebView2 (Windows) или GTK/WebKitGTK (Linux).")
		os.Exit(1)
	}
}
