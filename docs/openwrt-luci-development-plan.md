# OpenWrt LuCI development plan

Commit this plan separately. Remove it once the implementation passes acceptance and keep the user instructions in the READMEs.

Deliver two packages, a standalone installer, and GitHub Actions builds in this repository. The first target is OpenWrt 25.12 on `aarch64_cortex-a53`, built with the OpenWrt 25.12.5 `mediatek/filogic` SDK. Support fresh installations only. Keep the existing Go CLI and legacy manager available during the transition.

Use [ByeDPI-OpenWrt](https://github.com/DPITrickster/ByeDPI-OpenWrt) for the service package layout, [luci-app-byedpi](https://github.com/qb60/luci-app-byedpi) for the installer and separate autostart controls, and [tg-ws-proxy-rs](https://github.com/valnesfjord/tg-ws-proxy-rs/tree/main/openwrt) for the LuCI page layout.

## 1. Package the standalone service

Add the OpenWrt recipe under `openwrt/tg-ws-proxy-go/`. Install the binary at `/usr/bin/tg-ws-proxy-go`, the init script at `/etc/init.d/tg-ws-proxy-go`, and the UCI config at `/etc/config/tg-ws-proxy-go`. Declare the config as a conffile and give it restrictive permissions because it may contain credentials. The base package owns the service and its lifecycle and must install and run without LuCI.

**Check:** Inspect the package metadata and installed paths. Install the base package on a clean OpenWrt system and confirm it reports a running instance under procd.

## 2. Define the UCI settings and defaults

Use one named service section. Map values to the flags in `cmd/tg-ws-proxy/main.go`. Use UCI lists for DC mappings, upstream MTProto proxies, and custom Cloudflare domains. Keep SOCKS5 credentials, the MTProto secret, and upstream secrets out of status output and error messages.

| Setting | Fresh install |
| --- | --- |
| Mode | SOCKS5 |
| Listen address and port | `127.0.0.1:1080` |
| SOCKS5 credentials or MTProto secret | Unset |
| Cloudflare | Enabled, tried first, built-in domains |
| Cloudflare balancing | Off, as in the Go CLI |
| WebSocket pool size | 4 |
| Firewall mark and verbose logging | `0` and off |
| Custom DC mappings and upstream proxies | Unset |

For Cloudflare defaults pass `--cf-proxy --cf-proxy-first` and omit `--cf-domain`. Leave built-in domains and default DC mappings in the Go binary. Support the existing MTProto link address option alongside the MTProto settings.

**Check:** Capture the default command line and compare it with the table. Test a custom config for each mode, including repeatable flags and values with shell metacharacters.

## 3. Implement the procd lifecycle

Use a checked-in init script that loads UCI and passes individual arguments to procd. Enable autostart and start the service on the first live install of the base package. Respect offline package installation and preserve saved UCI values and the selected boot state on package upgrades. Let procd handle respawn and log stdout and stderr to the system journal.

Treat boot autostart and the current process as separate states. UCI apply reloads a running instance and leaves a stopped service stopped. A service with autostart disabled still supports an explicit start and restart. Validate settings through LuCI and enforce the same input requirements at the service or binary boundary for SSH users.

**Check:** Exercise install, start, stop, restart, reload, enable, and disable. Test that applying UCI changes to a stopped service leaves it stopped and that changing autostart leaves the current process alone.

## 4. Add the LuCI package and page

Add `openwrt/luci-app-tg-ws-proxy-go/` using the release-matched LuCI feed and `luci.mk`. Set package architecture to `all` and declare dependencies on `tg-ws-proxy-go` and `luci-base`. Register **Services → Telegram WS Proxy (Go)** in `menu.d`.

Use the standard LuCI JavaScript form components. Put process and autostart status at the top, settings below, and the system log on a separate tab. Show fields for the selected proxy mode. Expose **Start**, **Stop**, and **Restart** for the process and **Enable autostart** and **Disable autostart** for boot. Refresh status after actions and show failures in the page. Save & Apply writes the same UCI config used by the init script.

**Check:** Open the menu and verify both modes, the validation errors, separate process and boot indicators, and the effect of Save & Apply.

## 5. Restrict LuCI access

Use a small rpcd adapter for the service status, its recent system log entries, and the five fixed actions above. Limit it to this init script and return only the process and boot status needed by the page. Grant UCI access to this config only. Keep control methods under the write ACL and validate action names server-side.

**Check:** Verify that read access can view status and logs, while writes and service actions require write access. Reject unsupported actions. Check that status responses omit process arguments and credentials.

## 6. Build the packages in GitHub Actions

Add a build job for pull requests and manual runs. Use a one-row matrix with the OpenWrt release, SDK target, package architecture, and Go target. Pin the SDK archive and verify its SHA-256 using the [official SDK checksums](https://downloads.openwrt.org/releases/25.12.5/targets/mediatek/filogic/sha256sums). Build the static binary from the same checkout with Go 1.26, `CGO_ENABLED=0`, `GOOS=linux`, and `GOARCH=arm64`. Use the SDK to produce the two native OpenWrt `.apk` packages.

Upload the package pair as a CI artifact. Extend the existing tag release workflow to publish the same pair, `install.sh`, and SHA-256 checksums. Use deterministic asset names that identify the OpenWrt series, package version, and architecture. Keep the artifact naming and installer selection rules in sync.

**Check:** Inspect both APKs using the SDK tools. Verify package names, versions, architectures, dependency metadata, files, and conffile declaration. Verify that only tag releases publish to GitHub Releases.

## 7. Write the installer

Add a standalone POSIX shell `install.sh` at the repository root. Follow the [ByeDPI installer](https://github.com/qb60/luci-app-byedpi/blob/main/install.sh) for platform detection and release selection. Detect the OpenWrt series and package architecture, require `apk`, and select either the newest stable release with a matching pair or an explicitly supplied tag.

Download both packages and their checksums from the same release to a temporary directory. Verify both files before passing them together to `apk add`. If release packages are unsigned, limit `--allow-untrusted` to that invocation with the verified local files. Leave starting the service and enabling autostart to the base package.

**Check:** Use release metadata and download fixtures for success, explicit tag selection, unsupported platform, missing package, checksum mismatch, and package manager failure. Assert that download or verification failure occurs before any installation.

## 8. Verify the default Cloudflare route

In CI, verify that the fresh UCI config passes both Cloudflare flags and selects the built-in domain list. Reuse the controlled route test in `internal/socks5/handle_conn_test.go` to verify that Cloudflare is tried before direct WebSocket. Exercise data transfer using a fake WebSocket peer so CI does not depend on public endpoints.

On the router, send a Telegram message through SOCKS5, using an SSH tunnel to reach the loopback listener. Temporarily turn on verbose logging and confirm the Cloudflare route in the journal. Record this separately from the automated tests.

## 9. Run acceptance on OpenWrt

- [ ] Install the base package alone on a clean target and verify that SOCKS5 accepts connections immediately and that autostart is enabled.
- [ ] On a clean target, run `install.sh` and verify the two installed packages, the LuCI menu, and the initial running and autostart indicators.
- [ ] Exercise both proxy modes and verify that changing settings in LuCI applies them to the running service.
- [ ] Verify that Stop leaves autostart enabled, and that a reboot then starts the service. Disable autostart while it is running and verify that it stays running until stopped or rebooted. After reboot, verify that it stays stopped.
- [ ] Upgrade the package pair and verify that UCI settings and the selected autostart state survive.
- [ ] Verify a real Telegram connection through the default Cloudflare route as described above.

Record the firmware version, architecture, release tag, and the result of each check. Keep hardware checks pending until run on the target.

## 10. Update user documentation

Make `README.md` the English entry point and place `[Русский](README_ru.md)` at the top. Provide corresponding Russian instructions in `README_ru.md`. Keep both brief and limited to the supported platform, installation through `install.sh` or the two local packages, the LuCI menu, and what users need to connect and manage the service. Keep each shell command on one line when practical.
