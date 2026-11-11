# ProtosVPN Client — Stage 1 contract

## Scope
User approved implementation after accepting `/home/hermes/designs/protosvpn-classic/reference-v3.html` and `reference-v3.png`. Build a real import-first Windows/Linux client, not another static reference. No production VPN servers, routes, DNS, firewall or existing services may change. No fake connected state or network metrics. Real VPN engines/helper and OS credential-vault persistence are subsequent work.

Stack: Wails v2.17.1 (pinned maintained v2 release), Go, React + TypeScript + Vite. Go SDK at `/home/hermes/.local/toolchains/go/bin/go` (Go 1.27.3, official checksum verified). Module `protosvpn.local/client`. UI reference is the approved readable retro variant, not a literal Winamp player. All dependencies lockfile-pinned on installation.

## Ownership / layout
- Backend worker: `internal/client/**`, `cmd/preview/main.go`, Go tests. Do not modify root go.mod or frontend.
- Frontend worker: `frontend/**` only, including package.json, lockfile and frontend tests.
- Parent: root `go.mod`, `go.sum`, `main.go`, `wails.json`, documentation, packaging, browser integration and native builds.

## Service API (Go, package client)
`New() *Service`: in-memory service, cryptographically random internal IDs, no disk/secret persistence and no network engine. `NewHandler(service *Service) http.Handler` handles `/api/` routes. Preview `cmd/preview` binds **127.0.0.1:18761**, serves `frontend/dist` and delegates `/api/` to the same handler. In production Wails embeds frontend/dist and delegates missing asset paths to the same handler. No global CORS allow or arbitrary network listener.

### JSON contract
Profile DTO:
```
{id, name, protocol: "vless"|"openvpn", host, port, transport, security, requiresAuth, warnings: string[]}
```
No raw configuration, URI, UUID, key, certificate or password in DTOs/errors/diagnostics. Imported secret material stays private server-side RAM. No logs of request bodies, imported values or parse errors that echo input.

- `GET /api/state` -> `{version:"0.1.0", profiles: Profile[], selectedId:"", storageMode:"memory", csrfToken:"...", connection:{status:"disconnected",profileId:"",engineAvailable:false}}`. Return profiles in stable import order, never null. Select first imported profile by default. Native/preview state makes in-memory limitation explicit.
- `POST /api/import` body `{format:"vless"|"openvpn",content:string,name?:string}` -> HTTP201 `{profile:Profile}`.
- `POST /api/select` body `{id:string}` -> `{ok:true}`.
- `PATCH /api/profiles/{id}` body `{name:string}` -> `{profile:Profile}`. Rename metadata only, validated nonempty bounded label.
- `DELETE /api/profiles/{id}` -> `{ok:true}`. Remove private config too; select next existing or empty. No deletion outside memory.
- `POST /api/connect` body `{id:string}` -> HTTP409 `{error:{code:"ENGINE_UNAVAILABLE",message:"..."}}`; do not mutate state. `/api/disconnect` also honestly unavailable/no network work.
- `GET /api/diagnostics` -> `{version,os,arch,storageMode:"memory",engineAvailable:false,profileCount:number}` only safe allowlisted fields.
- Errors `{error:{code:string,message:string}}`, no reflected secrets. Invalid data400, unknown profile404, wrong method405. Limits 1MiB body/config (enforce actual bytes), label80 Unicode characters. Thread-safe service.

### Local security
Every mutation requires `X-ProtosVPN-Token` equal to bootstrap CSRF token (constant-time compare). Reject unsupported HTTP Host/Origin. Accept hosts only loopback (`127.0.0.1`, `localhost`, `::1`) and Wails virtual hosts (`wails.localhost`, `wails`); if present, Origin must be an exact corresponding local HTTP/Wails origin, never remote domains, suffix matches or null. No CORS reflection. GET responses no-store, nosniff. Never execute imported data, shell commands, scripts, remote fetches or subscriptions. POST errors do not disclose the token.

### Imports
VLESS: parse locally with strict vless scheme, UUID user id, no password, valid host/port, known security/transport; preserve exact private input server-side for future adapter; decode fragment into bounded display name, support escaped query values and IPv6; reject malformed/ambiguous fields without echoing URI. Import is syntax validation, not successful network authentication.
OpenVPN: parse a deliberately documented safe self-contained client subset; require remote host/port and tun client intent. Embedded `<ca>`, `<cert>`, `<key>`, `<tls-auth>`, `<tls-crypt>` treated as sensitive opaque blocks with matched closing tags. Reject external key/cert/auth files, script/plugin execution, includes, custom management listeners, logging/output paths and other executable/file directives. No expansion or file access. A no-argument `auth-user-pass` can set requiresAuth. Unknown/unsupported directives rejected explicitly without secret text. Certificate cryptographic validation is NOT claimed by this importer.

## Frontend
Use actual fetch API and the contract above. Bootstrap token from GET state and send header on mutations. No client secret/config persistence (no localStorage/sessionStorage/history for imports). Settings may persist UI-only preferences. Import textarea clears after success/cancel; accept file only by explicit browser picker. Render values as text, never dangerous HTML. No external fonts/analytics/resources.

Screens: approved main window with profile/status/empty traffic, real profiles table, import modal (`vless://` pasted or `.ovpn` file), rename, delete confirmation, settings (UI scaling only; absent network features disabled), diagnostics. Connect disabled with explicit helper/engine-unavailable explanation. No fake traffic, ping, connected status, DNS-protection or kill-switch assurances. Persistent unobtrusive notice: this initial build stores profiles in memory and doesn't connect yet. Startup empty state with import action. Keyboard accessible forms, Escape/cancel, readable errors, long names, narrow widths and scaling. Header/window buttons must not pretend to be functional if native runtime absent.

## Verification
TDD vertical slices. Backend real parser/service/httptest and concurrency tests. Frontend tests and full browser E2E with the real Go preview backend, synthetic nonfunctional credentials/TEST-NET hosts only. Verify import/rename/select/delete/reload semantics, unsupported config rejection, no secrets in DTO/diagnostics/localStorage/logs, no fake connection, auth/Origin guards, mobile overflow, desktop screenshot inspected by vision. Build Linux native and cross-compile Windows if environment permits; clearly distinguish compilation/headless startup from testing on real Windows. No claims of actual VPN connectivity in Stage1.
