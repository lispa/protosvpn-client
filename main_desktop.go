//go:build desktop && !native_smoke

package main

import (
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"io/fs"
	"os"
	"protosvpn.local/client/internal/client"
)

//go:embed all:frontend/dist
var embeddedFrontend embed.FS

func main() {
	assets, err := fs.Sub(embeddedFrontend, "frontend/dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Не удалось загрузить интерфейс ProtosVPN.")
		os.Exit(1)
	}
	service := client.New()
	if err = wails.Run(newDesktopOptions(assets, client.NewHandler(service))); err != nil {
		fmt.Fprintln(os.Stderr, "Не удалось запустить ProtosVPN. Проверьте наличие WebView2 (Windows) или GTK/WebKitGTK (Linux).")
		os.Exit(1)
	}
}
