//go:build desktop && native_smoke

// Test-only entrypoint. Never include native_smoke in distributable builds.
package main

import (
	"context"

	"encoding/json"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"io/fs"
	"net/http"
	"os"
	"protosvpn.local/client/internal/client"
	"sync"
	"testing/fstest"
)

func main() {
	var assets fs.FS = fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><html><head><title>Native API smoke test</title></head><body style='background:#202636;color:#ddd;font:20px monospace'><h1>Native API smoke test</h1><p>Test harness, not the ProtosVPN application interface.</p></body></html>")}}
	usingRealUI := false
	if path := os.Getenv("PROTOSVPN_SMOKE_ASSETS"); path != "" {
		assets = os.DirFS(path)
		usingRealUI = true
	}
	service := client.New()
	api := client.NewHandler(service)
	o := newDesktopOptions(assets, api)
	base := o.AssetServer.Middleware
	o.AssetServer.Middleware = func(next http.Handler) http.Handler {
		h := base(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/__native_smoke_result" {
				defer r.Body.Close()
				data, e := io.ReadAll(io.LimitReader(r.Body, 4097))
				var result map[string]any
				if e != nil || len(data) > 4096 || json.Unmarshal(data, &result) != nil {
					w.WriteHeader(400)
					return
				}
				result["usingRealUI"] = usingRealUI
				data, e = json.Marshal(result)
				if e == nil {
					e = os.WriteFile(os.Getenv("PROTOSVPN_SMOKE_REPORT"), data, 0600)
				}
				if e != nil {
					w.WriteHeader(500)
					return
				}
				w.WriteHeader(204)
				return
			}
			if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
				fmt.Printf("NATIVE_API %s %s Host=%q Origin=%q\n", r.Method, r.URL.Path, r.Host, r.Header.Get("Origin"))
			}
			h.ServeHTTP(w, r)
		})
	}
	var once sync.Once
	o.OnDomReady = func(ctx context.Context) {
		once.Do(func() {
			runtime.WindowExecJS(ctx, `(async()=>{let report={pass:false};try{const s=await(await fetch('/api/state')).json();const headers={'Content-Type':'application/json','X-ProtosVPN-Token':s.csrfToken};const i=await fetch('/api/import',{method:'POST',headers,body:JSON.stringify({format:'vless',content:'vless://11111111-1111-4111-8111-111111111111@198.51.100.20:443?security=tls&type=tcp#Native%20TEST'})});if(i.status!==201)throw Error('import '+i.status);const p=(await i.json()).profile;const rename=await fetch('/api/profiles/'+p.id,{method:'PATCH',headers,body:JSON.stringify({name:'Helsinki / TEST DATA'})});if(rename.status!==200)throw Error('rename '+rename.status);const connect=await fetch('/api/connect',{method:'POST',headers,body:JSON.stringify({id:p.id})});const st=await(await fetch('/api/state')).json();for(let n=0;n<30&&document.documentElement.lang!==st.systemLocale;n++)await new Promise(r=>setTimeout(r,50));const shell=document.querySelector('.app-shell').getBoundingClientRect();const chrome={controls:document.querySelectorAll('.window-controls button').length,drag:getComputedStyle(document.querySelector('.titlebar')).getPropertyValue('--wails-draggable').trim(),noDrag:getComputedStyle(document.querySelector('.window-controls button')).getPropertyValue('--wails-draggable').trim(),fillsWindow:shell.x===0&&shell.y===0&&Math.abs(shell.width-innerWidth)<1&&Math.abs(shell.height-innerHeight)<1};report={chrome,uiLanguage:document.documentElement.lang,systemLocale:st.systemLocale,pass:chrome.controls===3&&chrome.drag==='drag'&&chrome.noDrag==='no-drag'&&chrome.fillsWindow&&document.documentElement.lang===st.systemLocale&&connect.status===409&&st.profiles.length===1&&st.connection.status==='disconnected'&&st.storageMode==='memory',importStatus:i.status,renameStatus:rename.status,connectStatus:connect.status,profileCount:st.profiles.length,connection:st.connection.status};}catch(e){report={pass:false,error:String(e)}}await fetch('/__native_smoke_result',{method:'POST',body:JSON.stringify(report)});if(report.pass)location.reload()})()`)
		})
	}
	if e := wails.Run(o); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
