# Verification — ProtosVPN 0.2.3-preview

## Profile persistence

The desktop service now uses per-user persistent storage rather than an in-memory-only profile list. Import, rename, delete and selection save before publishing their changes. The vault is outside the executable directory, so moving a portable build does not change the storage location.

- Windows: `%APPDATA%\ProtosVPN\profiles.vault`, protected by current-user DPAPI and a restricted directory ACL. No plaintext fallback.
- Linux: the user configuration directory (`$XDG_CONFIG_HOME/ProtosVPN` or `~/.config/ProtosVPN`), AES-GCM with a local permission-protected key. This is not an OS keyring; copying both files exposes the key.
- Load/save failures are shown explicitly. A failed load blocks mutations instead of overwriting an unreadable vault. Failed saves do not silently apply a mutation.
- Existing memory-only versions have no saved profiles to migrate. Import once in this version. Compatibility consent and active connections are not persisted.

## Measured dashboard

Session time is based on the connection start timestamp. Upload/download totals and rates come from the actual Xray outbound counters, not total-device traffic. The HTTPS-check duration is the initial connectivity check, not ICMP ping or continuous availability monitoring. Unavailable measurements display a dash. The unused traffic graph is explicitly identified as unavailable.

Both official cores were exercised against a controlled local VLESS server with HTTPS upload/download; counters increased and reads did not reset them. Tests covered cancellation, bounded/sanitized stats output, worker message validation and rate calculations.

## Executed checks

- `go test -race -count=1 ./...` and `go vet ./...` passed on the Linux build host.
- Real-core transport tests passed for TCP/raw, WebSocket, gRPC, HTTP Upgrade and XHTTP, including a separate run using the legacy core.
- Real Xray statistics integration passed for modern v26.11.9 and legacy v25.12.8.
- 71 frontend tests passed; TypeScript checking and production frontend build passed.
- Eight installer/packaging tests passed, including tampered legacy core rejection.
- Windows amd64 Wails production cross-build passed. Ubuntu 24.04 production build and desktop-tag tests passed in the build container.
- NSIS installer compilation passed. Artifact checksums and packaged payloads are checked during release preparation.

## Compatibility and limits

Modern official Xray v26.11.9 remains the default. Legacy v25.12.8 is selected only after explicit opt-in and confirmation for VLESS without TLS/Reality. There is no automatic fallback. Modern Xray rejects the tested public-destination plaintext VLESS configuration; the legacy core accepts it. Old cores may lack security fixes. VLESS without TLS/Reality provides no transport encryption; HTTPS websites retain their own TLS.

The user confirmed that 0.2.2 changed their browser's public IP when connected and restored it after Disconnect. That observation does not verify 0.2.3 profile persistence, DPAPI, metrics or installer behavior on Windows. Native Windows execution was not available on the build host; these still require checking on Windows. Builds are unsigned.

System HTTP proxy is not a full-device VPN: applications may bypass it, there is no kill switch, Linux is management-only and OpenVPN connectivity is not implemented. The restoration helper cannot cover forced termination of both helper and GUI, power loss or reboot. If necessary, disable the owned manual loopback proxy in Windows network proxy settings or restore your original settings. Do not run competing proxy clients.
