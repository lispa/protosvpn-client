# ProtosVPN

A desktop client for managing your own VLESS and OpenVPN profiles, with an English/Russian interface.

On **Windows**, connect to VLESS servers using the bundled Xray core and the system HTTP proxy. Optional **Compatibility mode** uses an older core for unencrypted VLESS, with an explicit risk warning and confirmation. Connect checks HTTPS connectivity through the server before enabling the proxy; Disconnect restores the previous proxy settings.

**This is not a full-device VPN:** apps that ignore the system proxy may connect directly. There is no kill switch. OpenVPN connections and Linux connections are not implemented yet. Profiles are saved in protected per-user storage and restored on restart.

## Download — 0.2.3-preview

| Platform | Download |
| --- | --- |
| Windows x64 — installer | [Setup.exe](https://github.com/lispa/protosvpn-client/releases/download/v0.2.3-preview/ProtosVPN-0.2.3-preview-windows-amd64-setup.exe) |
| Windows x64 — portable | [ZIP](https://github.com/lispa/protosvpn-client/releases/download/v0.2.3-preview/ProtosVPN-0.2.3-preview-windows-amd64.zip) |
| Linux x64 — profile management only | [Ubuntu 24.04 tar.gz](https://github.com/lispa/protosvpn-client/releases/download/v0.2.3-preview/ProtosVPN-0.2.3-preview-ubuntu24-amd64.tar.gz) |

**Windows:** WebView2 Runtime is required. The installer creates Start menu and Desktop shortcuts. Keep the `core` folder when using the portable version. Disconnect and close the old client before updating.

**Profiles:** Import once after upgrading from memory-only versions. Profiles and selection are restored on restart; compatibility consent is not saved. Windows uses current-user DPAPI in `%APPDATA%\ProtosVPN`. Linux uses encrypted storage with a local permission-protected key, not an OS keyring. A storage error is displayed instead of silently replacing unreadable data.

**Dashboard:** Shows measured proxy traffic, session time and the initial HTTPS-check duration, not device-wide traffic or ICMP ping.

**Linux:** Requires GTK3 and WebKitGTK 4.1.

Experimental, unsigned builds. Real Windows behaviour and your external server still need testing. See [release notes](https://github.com/lispa/protosvpn-client/releases/tag/v0.2.3-preview) for verification details and proxy recovery instructions.
