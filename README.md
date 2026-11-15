# ProtosVPN

A desktop client for your own VLESS and OpenVPN profiles, with an English/Russian interface.

**Windows preview:** VLESS connections use the current user's system HTTP proxy. Connect starts the bundled Xray core, checks HTTPS connectivity through the server and enables the proxy. Disconnect restores the previous proxy settings. Apps that ignore the Windows system proxy can connect directly. **This is not a full-device VPN or a kill switch.**

OpenVPN connections and Linux system-proxy connections are not implemented yet. Profiles are stored in memory and disappear when the application closes.

## Download

| Platform | Download |
| --- | --- |
| Windows x64 — installer | [Download Setup.exe](https://github.com/lispa/protosvpn-client/releases/download/v0.2.0-preview/ProtosVPN-0.2.0-preview-windows-amd64-setup.exe) |
| Windows x64 — portable | [Download ZIP](https://github.com/lispa/protosvpn-client/releases/download/v0.2.0-preview/ProtosVPN-0.2.0-preview-windows-amd64.zip) |
| Linux x64 — profile management only | [Download tar.gz](https://github.com/lispa/protosvpn-client/releases/download/v0.2.0-preview/ProtosVPN-0.2.0-preview-ubuntu24-amd64.tar.gz) |

**Windows:** Close the old client before updating. The installer creates Start menu and Desktop shortcuts. WebView2 Runtime is required; unsigned builds may trigger SmartScreen. For portable use, keep the `core` folder next to `ProtosVPN.exe`.

**Recovery:** If a forced shutdown leaves a broken proxy, open Windows **Settings → Network & Internet → Proxy** and disable ProtosVPN's `127.0.0.1` manual proxy (or restore your own previous settings). Do not run another proxy client at the same time. This preview still needs real Windows installation and network testing.

**Linux:** Ubuntu 24.04, GTK3 and WebKitGTK 4.1 are required. Extract the archive and run `ProtosVPN`.
