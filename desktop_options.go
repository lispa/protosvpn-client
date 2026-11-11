//go:build desktop

package main

import (
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"io/fs"
	"net/http"
)

func newDesktopOptions(assets fs.FS, api http.Handler) *options.App {
	return &options.App{
		Title: "ProtosVPN", Width: 940, Height: 820, MinWidth: 640, MinHeight: 640,
		Frameless:        true,
		DisableResize:    false,
		BackgroundColour: options.NewRGB(20, 24, 33),
		AssetServer:      &assetserver.Options{Assets: assets, Middleware: newAPIMiddleware(api)},
		LogLevel:         logger.ERROR, LogLevelProduction: logger.ERROR,
	}
}
