# Verification of development preview 0.1.2

## Frameless window update: re-executed checks

- `go test -race ./...` and `go vet ./...`: PASS.
- Frontend: 18 tests, including native control wiring and translated labels: PASS; TypeScript and Vite build: PASS.
- Browser native-layout check with stubbed window runtime: full-window shell at 640x640, 940x820 and 1440x1000; no outer padding or horizontal overflow; drag/no-drag styles; minimise/maximise/quit call wiring; modal Escape: PASS. This check does not exercise an OS window manager.
- Language and complete application Playwright smoke tests against real Go API: PASS.
- Linux Wails build and desktop configuration tests: PASS. Configuration requires Frameless=true and DisableResize=false.
- Native WebKit + real Go API smoke: PASS; three titlebar controls, full-window shell and Wails drag/no-drag CSS confirmed alongside Russian locale and import/rename/refused-connect checks.
- Exact production Linux executable separately started and captured under Xvfb: PASS; custom titlebar visible, no standard titlebar or internal outer backdrop.
- Windows GUI x64 cross-build with desktop tag: PASS. Real Windows launch, OS drag/resize, minimise/maximise/restore behaviour still require user verification. Xvfb has no normal desktop window manager; no claim of end-to-end window-manager interaction testing is made.

## Previous localization verification (0.1.1)

## Localization update: re-executed checks

- `go test -race ./...` and `go vet ./...`: PASS.
- Frontend: 15 tests (9 language + 6 application): PASS; TypeScript and Vite production build: PASS.
- Real Playwright browser + Go API: en-US, ru-RU, fi-FI fallback; manual language switching, persistence on reload and restoration of system language: PASS.
- Full existing real browser import/profile/settings/diagnostics smoke: PASS.
- Linux production binary rebuilt; desktop Go tests in Ubuntu 24.04 container: PASS.
- Native WebKit smoke under Russian system locale, real assets and Go API: PASS; report confirms `systemLocale=ru`, `uiLanguage=ru`, import 201, rename 200, connect correctly refused 409.
- Exact updated Linux distributable separately launched under Xvfb with Russian locale and captured.
- Updated Windows x64 GUI executable cross-built with desktop tag and WebView2 error mode; PE32+ format verified. Not executed on Windows.
- Native smoke runs use an isolated network-disabled container; its sandbox override is test-only.

## Previous baseline checks (not all re-executed for this update)

## Real executions performed

- Go race tests: `go test -race ./internal/client ./cmd/preview` — PASS.
- Go middleware tests: `go test .` — PASS.
- `go vet ./internal/client ./cmd/preview` — PASS.
- Desktop configuration/middleware tests inside Ubuntu 24.04 GTK/WebKit container: `go test -tags desktop,webkit2_41 .` — PASS.
- Frontend: 6 Vitest tests — PASS; TypeScript typecheck and Vite production build — PASS.
- Real standalone Go HTTP process: VLESS and inline OpenVPN import, select/rename/delete, redacted metadata, unsafe configuration rejection, missing CSRF and invalid Host/Origin rejection, and unavailable-engine response — PASS.
- Playwright against the real built frontend and Go HTTP API, **without route mocks**: VLESS paste and OpenVPN file import, invalid import, rename/select/delete/cancel, import input clearing, non-secret settings persistence, diagnostics, reload, no page errors, narrow-width overflow assertion — PASS.
- Native Wails/WebKitGTK integration on Linux: real local API fetches from native WebView, import 201, rename 200, connect refused 409, disconnected state, memory storage and actual React assets (`usingRealUI=true`) — PASS. Synthetic profiles only.
- **Exact distributable Linux executable** was separately started under Xvfb, remained running and rendered its embedded React UI and actual empty Go service state. This is not merely a mockup or browser preview.
- Windows executable: successful Wails cross-build and PE32+ GUI x86-64 format inspection. **Not run on a real Windows machine.**
- `govulncheck -test ./internal/client/... ./cmd/preview/...`: no vulnerabilities found in that checked scope.
- `npm audit --omit=dev`: 0 reported production dependency vulnerabilities.

These checks are not a full security audit, a certificate validation test, or proof of a working VPN connection. No actual VPN connection exists in this increment.

## Headless test isolation

Builder image installed GTK/WebKit only inside Docker; no host GTK/WebKit packages or VPN server settings were changed. Native tests ran as the unprivileged host UID inside a container with `--network none`. WebKit sandbox disabling was restricted to that disposable headless test environment. It is not embedded in the executable or recommended for normal use. Host font files and passwd/group mappings were mounted read-only for X11/DBus test setup; test outputs alone were writable.

The `native_smoke` build tag is **test-only** and excluded from both distributed binaries. Its synthetic profile and report endpoint are never included in the normal desktop entrypoint.

## Remaining limits

- Configurations and profiles only survive while the Go app process runs; there is no credential vault or file persistence.
- OpenVPN importer intentionally supports only the documented self-contained safe subset, not arbitrary OpenVPN exports.
- No OpenVPN/sing-box core, privileged helper, TUN, DNS/route management, kill switch, auth password prompt, update mechanism, signing or installer.
- Linux binary targets the tested Ubuntu 24.04 + GTK3/WebKitGTK4.1 environment; no all-distro portability claim.
- Windows binary is unsigned and requires Microsoft WebView2 Runtime.

`release/SHA256SUMS.txt` records the produced binary and archive checksums. Release archives contain code/configuration previews only, never user VPN credentials. UI screenshots marked TEST DATA contain synthetic reserved-address fixtures; the exact production startup screenshot contains no profiles.
