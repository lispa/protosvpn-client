# proxyengine

Single-use owner of a trusted **official Xray** subprocess. No dependency on
`internal/client` and no shell execution. Production configuration is passed only
through the child's stdin (`xray run -config stdin:`); stdout/stderr are discarded
and all returned errors are static. The core logging configuration disables both
access and error logging. Windows children use `HideWindow` and `CREATE_NO_WINDOW`.

```go
engine := proxyengine.New(absoluteTrustedCorePath)
defer engine.Close()
endpoint, err := engine.Start(startupContext, vlessURI)
// endpoint is 127.0.0.1:PORT (HTTP proxy, including CONNECT), not a URL.
// After Start succeeds, cancelling startupContext does not stop the core.
// Watch engine.Done() to restore system proxy settings on unexpected exit.
if err == nil {
    err = proxyengine.Check(checkContext, endpoint, "https://www.gstatic.com/generate_204")
}
```

`Start` has a ten-second upper bound. Readiness checks Xray's HTTP proxy response
(origin-form requests must return HTTP 400 and `Proxy-Connection: close`), not just
an open TCP socket. **It does not authenticate the VLESS server or prove internet
access.** The separate `Check` has a twenty-second upper bound, uses the specified
proxy without environment/direct fallback, checks ordinary HTTPS certificates,
rejects redirects and non-2xx responses, and never returns target details.

Supported: canonical UUID VLESS IDs, explicit server port, security none/TLS/Reality,
RAW/TCP, WebSocket, gRPC, HTTPUpgrade and XHTTP. TLS certificate verification cannot
be disabled. Reality requires explicit SNI, supported fingerprint and a nonzero
32-byte base64url public key; short IDs must be even-length hexadecimal up to eight
bytes. Vision is restricted to RAW plus TLS/Reality. Parameters are allowlisted
per security/transport; duplicates, unknown parameters, unsupported modes and
cross-transport parameters fail closed. Legacy HTTP/H2 transport is not XHTTP and
is deliberately rejected. This is a conservative supported subset, not a generic
Xray JSON importer. Invalid startup consumes the engine, too.

The HTTP proxy listens only on IPv4 loopback and has no authentication; like other
local desktop proxies, it assumes other local processes are trusted. Port
allocation releases a temporary reservation before Xray binds (Xray has no
inherited-listener interface); bind failures abort startup. `Close` kills and
reaps the owned child and is concurrency-safe/idempotent. The caller owns system
proxy installation/restoration and must monitor `Done`.

## Verification

Run `go test -race ./internal/proxyengine`. Integration tests use the actual binary
at `build/cores/GOOS-GOARCH/xray[.exe]`, or `XRAY_TEST_BINARY`. Without a core those
tests explicitly skip. Real local VLESS servers exercise all five transports,
HTTP traffic, HTTPS CONNECT certificate rejection, startup-context independence,
unexpected process exit, and cleanup. Configuration variants are checked using
`xray run -test -config stdin:`. No secret configuration files are written.

Xray 26.11.9 blocks private-address targets by default for VLESS-to-freedom.
The **test server only** explicitly permits `127.0.0.1/32` using freedom
`finalRules`; production client configuration has no direct/freedom outbound or
such bypass. External gstatic availability is intentionally not required by tests.
