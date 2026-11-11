//go:build desktop

package main

import (
	"github.com/wailsapp/wails/v2/pkg/logger"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestDesktopUsesReviewedAssetsAndRestrictedAPI(t *testing.T) {
	files := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("reviewed-ui")}}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		io.WriteString(w, "engine-unavailable")
	})
	o := newDesktopOptions(files, api)
	if o.Title != "ProtosVPN" {
		t.Fatalf("wrong window identity: %q", o.Title)
	}
	if o.Width < o.MinWidth || o.Height < o.MinHeight || o.MinWidth < 640 || o.MinHeight < 600 {
		t.Fatal("unreadable window bounds")
	}
	if !o.Frameless || o.DisableResize {
		t.Fatal("custom window chrome must be frameless and resizable")
	}
	if o.AssetServer == nil || o.AssetServer.Assets == nil || o.AssetServer.Middleware == nil {
		t.Fatal("missing local assets/API routing")
	}
	if o.LogLevelProduction != logger.ERROR {
		t.Fatal("production request debugging must not be enabled")
	}
	if len(o.Bind) != 0 {
		t.Fatal("must not expose extra native RPC methods")
	}
	f, e := o.AssetServer.Assets.Open("index.html")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	if string(data) != "reviewed-ui" {
		t.Fatal("asset source changed")
	}
	h := o.AssetServer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("API reached assets") }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "http://wails.localhost/api/connect", nil))
	if w.Code != 409 {
		t.Fatal("engine status masked")
	}
}
