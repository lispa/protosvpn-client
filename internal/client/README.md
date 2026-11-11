# Stage 1 client backend

`client.New()` owns only process memory. `client.NewHandler(service)` implements the Stage 1 JSON API. Preview is `go run ./cmd/preview` from the workspace root, after building `frontend/dist`; its address is fixed at **127.0.0.1:18761**. There are no external backend dependencies, VPN subprocesses, subscription requests, disk persistence or operating-system networking changes.

## Safe imports

Imports validate syntax, not authentication, connectivity or certificate cryptography. Exact input remains in private backend memory, separate from metadata DTOs. State and diagnostics never contain URI user IDs, raw configurations, keys or certificates. Delete removes the service's references to private configuration; Go-managed strings are **not** a secure-memory-erasure facility. No claims are made about swap/core-dump protection or OS credential vaults.

### VLESS

Requires `vless://`, a UUID user ID without password, explicit port 1–65535, and DNS/IP host (IPv6 authority must be bracketed). Decodes the fragment as a display label, up to 80 Unicode characters; no control characters. Optional override name has the same bound. Defaults: transport `tcp`, security `none`.

Supported transport labels: `tcp`, `ws`, `grpc`, `http`, `httpupgrade`, `xhttp`. Security labels: `none`, `tls`, `reality`. Recognized query fields: `security`, `type`, `encryption` (only `none`), `flow`, `sni`, `fp`, `pbk`, `sid`, `spx`, `path`, `host`, `serviceName`, `alpn`, `headerType`, `authority`, `mode`. Query escaping is decoded; duplicate/unknown fields and malformed encoding are rejected. Recognizing an adapter option is **not** a claim that its semantic engine requirements are satisfied. No engine is present.

### OpenVPN supported subset

One directive per line; blank lines and full-line `#`/`;` comments allowed; CRLF accepted. Quoting, line continuations, environment expansion and inline comments are deliberately unsupported. Duplicate directives/blocks are rejected. Required:

- `client`
- `dev tun`
- Exactly one `remote <DNS-or-IP-host> <port>` (port 1–65535; bare IPv6 accepted in this whitespace-delimited host field)

Optional:

- `proto udp|udp4|udp6|tcp-client|tcp4-client|tcp6-client` (default `udp`)
- No-argument `auth-user-pass` marks `requiresAuth`; this build neither prompts nor stores passwords
- No-argument `nobind`, `persist-key`, `persist-tun`, `remote-random`, `pull`
- `resolv-retry infinite`, `remote-cert-tls server`, `key-direction 0|1`
- `cipher` / `data-ciphers-fallback`: AES-128/256-GCM, AES-128/256-CBC, CHACHA20-POLY1305
- `data-ciphers`: colon-delimited list of those cipher names
- `auth SHA256|SHA384|SHA512|SHA1`, `tls-version-min 1.2|1.3`, `verb 0|1|2|3|4`
- Nonempty inline `<ca>`, `<cert>`, `<key>`, `<tls-auth>`, `<tls-crypt>` blocks, with exact matching closing tag on its own line. Contents are sensitive **opaque text**, not cryptographically verified. Nested/mismatched tags are rejected.

Everything else is rejected, including external cert/key/auth files, `config`, script hooks, `plugin`, custom management listeners, output/logging paths, routing/DNS overrides and unknown directives. No filesystem access, fetching or execution occurs during import. This is intentionally not a general OpenVPN configuration parser, and accepted profiles are never passed to an engine in Stage 1. More permissive engine integration requires a separate reviewed policy.

## API security

Bootstrap `GET /api/state`, then send the returned `csrfToken` in `X-ProtosVPN-Token` on every mutation. Token comparison is constant-time. Host allowlist is loopback plus Wails virtual hosts. A supplied Origin must match the exact request authority and a permitted local HTTP/Wails scheme; remote/null Origins and cross-site fetch metadata are denied. Forwarded headers are not trusted. No CORS reflection. Responses are no-store/nosniff. JSON/body errors use static messages; raw parse errors and bodies are never logged. Body and decoded configuration limits are each 1 MiB **actual bytes**, not Content-Length alone.

VPN engine/system helper are absent. Connect/disconnect report `ENGINE_UNAVAILABLE` (409) without changing connection/selection state; unknown profile operations report 404. Diagnostics contain only version, OS, architecture, memory storage mode, absent engine and profile count.

## Verification

```sh
export PATH=/home/hermes/.local/toolchains/go/bin:$PATH GOTOOLCHAIN=local GOTELEMETRY=off
go test -race ./internal/client ./cmd/preview
go vet ./internal/client ./cmd/preview
```

Tests use explicit synthetic credentials, TEST-NET IPs and `.test` domains. They exercise real parsers, CRUD, independent snapshots, concurrent access, API lifecycles, Host/Origin/token failures, limits and preview routing. No test requires a live VPN server or alters networking.
