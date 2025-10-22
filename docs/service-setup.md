# GoTray system context deployment

This guide explains how to run GoTray as a privileged background service while keeping the tray icon available in every signed-in user session. The service owns the encrypted configuration file; user-session agents authenticate through a shared token before rendering the menu.

## 1. Prepare shared configuration

1. Copy `.env.example` to `/etc/gotray/.env` (or another secure location) and set the following variables:
   - `GOTRAY_SECRET`: strong encryption passphrase (stored only on the server).
   - `GOTRAY_SERVICE_TOKEN`: optional explicit IPC token. Omit to derive a token from `GOTRAY_SECRET`.
   - `GOTRAY_SERVICE_ADDR`: loopback endpoint used for IPC (defaults to `127.0.0.1:47863`).
2. Ensure the encrypted configuration path (`GOTRAY_CONFIG_PATH`) lives in a directory writable by the service account (for Linux this defaults to `/var/lib/gotray`).

## 2. Linux (systemd)

1. Run `scripts/install.sh` as root. The script builds the binary into `/opt/gotray`, creates a locked-down `gotray` system account, installs a `gotray.service` systemd unit, and seeds `/var/lib/gotray/config.enc`.
2. Enable and start the service: `sudo systemctl enable --now gotray.service` (handled by the script).
3. For each desktop user, launch the tray agent at login (GNOME/KDE autostart, XDG autostart entry, etc.):
   ```bash
   /opt/gotray/gotray tray
   ```
   The agent reads `GOTRAY_SERVICE_TOKEN` from `/etc/gotray/.env` by default; ensure the file is world-readable only if needed or copy minimal variables into a user-specific wrapper script.

## 3. Windows (Service Control Manager + per-user agent)

1. Compile the binary and place it in `C:\Program Files\GoTray`.
2. Register the system service from an elevated PowerShell prompt:
   ```powershell
   $env:GOTRAY_SECRET = "<strong passphrase>"
   $env:GOTRAY_SERVICE_ADDR = "127.0.0.1:47863"
   $env:GOTRAY_SERVICE_TOKEN = "<shared token>"
   New-Service -Name GoTray -BinaryPathName '"C:\Program Files\GoTray\gotray.exe" serve' -DisplayName 'GoTray Service' -Description 'GoTray system context service' -StartupType Automatic
   Start-Service GoTray
   ```
3. Deploy the tray agent via a user logon script or Group Policy Run entry:
   ```powershell
   Start-Process -FilePath 'C:\Program Files\GoTray\gotray.exe' -ArgumentList 'tray' -WindowStyle Hidden
   ```
   The agent uses the same environment variables; store them in `C:\ProgramData\GoTray\gotray.env` and reference them from a small wrapper script if you prefer not to expose them directly.

## 4. macOS (launchd)

1. Build the binary and copy it to `/usr/local/bin/gotray`.
2. Create a LaunchDaemon for the system service (`/Library/LaunchDaemons/com.example.gotray.plist`):
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
   <plist version="1.0">
     <dict>
       <key>Label</key><string>com.example.gotray</string>
       <key>ProgramArguments</key>
       <array>
         <string>/usr/local/bin/gotray</string>
         <string>serve</string>
       </array>
       <key>EnvironmentVariables</key>
       <dict>
         <key>GOTRAY_SECRET</key><string>__strong_passphrase__</string>
         <key>GOTRAY_SERVICE_TOKEN</key><string>__shared_token__</string>
         <key>GOTRAY_SERVICE_ADDR</key><string>127.0.0.1:47863</string>
       </dict>
       <key>RunAtLoad</key><true/>
       <key>KeepAlive</key><true/>
     </dict>
   </plist>
   ```
3. Load the daemon: `sudo launchctl load /Library/LaunchDaemons/com.example.gotray.plist`.
4. For each user, install a LaunchAgent (`~/Library/LaunchAgents/com.example.gotray.tray.plist`) with `ProgramArguments` set to `gotray tray` and environment variables referencing the shared token. Load it using `launchctl load ~/Library/LaunchAgents/com.example.gotray.tray.plist`.

## 5. Monitoring and troubleshooting

* The system service logs connection attempts and synchronization activity to stdout/stderr. Redirect this output using systemd's journal, Windows Event Viewer (via `sc.exe failure` logging), or `launchctl` logs on macOS.
* Ensure firewalls allow loopback connections on the configured `GOTRAY_SERVICE_ADDR`.
* If tray agents report `unauthorized`, confirm they are using the same `GOTRAY_SERVICE_TOKEN` as the service.

## 6. Development sandbox

Use `scripts/install_dev.sh` to deploy a secondary service alongside production. It installs its assets into `/opt/gotray-dev`, reads environment variables from `/etc/gotray/dev.env`, and keeps data in `/var/lib/gotray-dev/config.enc` so production data remains untouched.
