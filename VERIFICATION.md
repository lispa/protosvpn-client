# Verification — ProtosVPN 0.2.2-preview

## Reproduced incompatibility

The user ran the bundled Windows Xray v26.11.9 successfully, validated minimal stdin JSON (`Configuration OK`), and started a loopback HTTP inbound. Their curl probe returned HTTP 400 with `Proxy-Connection: close`, matching our readiness test. This establishes the manual executable/stdin/inbound path, not the app-generated VLESS config or an external connection.

On this Linux host, the actual modern Xray rejected a synthetic VLESS `security=none` / `encryption=none` outbound to a non-private destination during configuration loading:

> vless without TLS or other encryption is prohibited unless the server address is a private IP or domain

The same policy is present in the pinned upstream `infra/conf/xray.go`. Existing loopback-only integration fixtures did not expose the public-destination restriction. The user's profile displayed TCP / NONE, consistent with this incompatibility, but the actual private URI was neither requested nor tested.

## Explicit compatibility mode

The user explicitly approved an older official core for the existing unencrypted scheme. Modern v26.11.9 remains the default. Legacy v25.12.8 is a separately named payload, selected only by an explicit compatibility request. Missing legacy payload must fail, never fall back. Compatibility is restricted to VLESS without TLS/Reality; no TLS validation flags or upstream connectivity checks are weakened.

The frontend requires an opt-in and warning/confirmation before sending the compatibility flag. Consent is session-only and reset by profile selection; an active risk warning is driven by actual connection metadata. Old-core risks and the absence of VLESS-layer encryption are stated; HTTPS retains its independent TLS.

## Executed evidence

- Go API opt-in regression was observed failing before implementation, then passed. Normal Connect never calls the compatibility connector; explicit compatibility selection is represented in authoritative state.
- Worker rejects TLS compatibility, propagates the explicit variant to engine selection, and reports missing legacy binaries without automatic fallback. Unit/race checks passed.
- Offline production path selection test with the real official cores: modern failed public-plaintext startup as expected; explicit legacy became ready. No external target connection was made by that test.
- Full existing real-core transport integration suite executed against legacy v25.12.8: raw TCP, WS, gRPC, HTTP Upgrade and XHTTP reached a controlled local HTTP server. Untrusted local TLS certificates remained rejected; core cancellation/exit lifecycle checks passed.

Final frontend, package and full-suite checks are recorded in the release notes after execution.

## Limits

No Windows execution was available on the build host; native WinINet enable/restore and the user's actual server connection still require testing on their machine. The user evidence above is from 0.2.1 and is not evidence that 0.2.2 connects. Builds are unsigned. System HTTP proxy is not a full-device VPN; applications may bypass it, there is no kill switch, Linux is management-only, OpenVPN connectivity is not implemented, and profiles are memory-only.

An in-memory restoration helper cannot cover forced termination of both helper and GUI, power loss or reboot. If necessary, disable the owned manual loopback proxy in Windows network proxy settings or restore your original settings. Do not run competing proxy clients.
