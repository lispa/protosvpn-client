# ProtosVPN

A desktop client for managing your own VLESS and OpenVPN profiles, with an English/Russian interface.

On **Windows**, connect to VLESS servers using the bundled Xray core and the system HTTP proxy. Connect checks HTTPS connectivity through the server before enabling the proxy; Disconnect restores the previous proxy settings.

**This is not a full-device VPN:** apps that ignore the system proxy may connect directly. There is no kill switch. OpenVPN connections and Linux connections are not implemented yet. Profiles are stored in memory only.

## Download — 0.2.0-preview

| Platform | Download |
| --- | --- |
| Windows x64 — installer | [Setup.exe](https://github.com/lispa/protosvpn-client/releases/download/v0.2.0-preview/ProtosVPN-0.2.0-preview-windows-amd64-setup.exe) |
| Windows x64 — portable | [ZIP](https://github.com/lispa/protosvpn-client/releases/download/v0.2.0-preview/ProtosVPN-0.2.0-preview-windows-amd64.zip) |
| Linux x64 — profile management only | [Ubuntu 24.04 tar.gz](https://github.com/lispa/protosvpn-client/releases/download/v0.2.0-preview/ProtosVPN-0.2.0-preview-ubuntu24-amd64.tar.gz) |

**Windows:** WebView2 Runtime is required. The installer creates Start menu and Desktop shortcuts. Keep the `core` folder when using the portable version. Disconnect and close the old client before updating.

**Linux:** Requires GTK3 and WebKitGTK 4.1.

Experimental, unsigned builds. Real Windows behaviour and your external server still need testing. See [release notes](https://github.com/lispa/protosvpn-client/releases/tag/v0.2.0-preview) for verification details and proxy recovery instructions.
