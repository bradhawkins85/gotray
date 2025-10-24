# GoTray per-user deployment

This guide explains how to run GoTray as a stand-alone tray application for each desktop user. Every instance reads the encrypted configuration stored in the invoking user's profile and refreshes the menu directly from disk—no background system service or IPC channel is required.

## 1. Prepare the environment

1. Copy `.env.example` to a secure location and set the following variables:
   - `GOTRAY_SECRET`: strong encryption passphrase. Each user should choose a unique value.
   - `GOTRAY_CONFIG_PATH` (optional): override the default configuration path. By default the encrypted file lives in `~/.config/gotray/config.enc`.
2. Ensure the destination directory for `GOTRAY_CONFIG_PATH` exists and is writable only by the target user.
3. Build the binary:
   ```bash
   go build -o gotray ./cmd/gotray
   ```

## 2. Linux (systemd per-user service)

The production installer provisions a systemd template that launches GoTray as the target desktop user.

1. Run the installer as root, passing the username that should own the tray session:
   ```bash
   sudo GOTRAY_INSTALL_USER=<username> ./scripts/install.sh
   ```
   The script will:
   - Build `/opt/gotray/gotray`.
   - Copy a user-specific environment file to `/etc/gotray/<username>.env`.
   - Create `/var/lib/gotray/<username>/config.enc` (owned by the user).
   - Install `/etc/systemd/system/gotray@.service` with `ExecStart=/opt/gotray/gotray run`.
2. Enable and start the user service:
   ```bash
   sudo systemctl enable --now "gotray@<username>.service"
   ```
3. The tray will now launch automatically at boot for that user. Use `systemctl status gotray@<username>.service` to confirm it is running.

To install for additional users, repeat the process with a different `GOTRAY_INSTALL_USER`. Each user receives an isolated configuration directory and environment file.

## 3. Windows (per-user Scheduled Task)

1. Build the binary and copy it to `C:\Program Files\GoTray\gotray.exe`.
2. Open Task Scheduler and create a new task that runs `gotray.exe run` at logon for the desired user.
3. Define the following user environment variables before launching the task:
   - `GOTRAY_SECRET`
   - Optional `GOTRAY_CONFIG_PATH`
4. On first run the tray seeds the configuration and stores it in `%APPDATA%\GoTray\config.enc` unless `GOTRAY_CONFIG_PATH` overrides the location.

## 4. macOS (LaunchAgent)

1. Place the binary at `/usr/local/bin/gotray`.
2. Create `~/Library/LaunchAgents/com.example.gotray.plist` with the contents below:
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
   <plist version="1.0">
     <dict>
       <key>Label</key><string>com.example.gotray</string>
       <key>ProgramArguments</key>
       <array>
         <string>/usr/local/bin/gotray</string>
         <string>run</string>
       </array>
       <key>EnvironmentVariables</key>
       <dict>
         <key>GOTRAY_SECRET</key><string>__strong_passphrase__</string>
       </dict>
       <key>RunAtLoad</key><true/>
       <key>KeepAlive</key><true/>
     </dict>
   </plist>
   ```
3. Load the agent: `launchctl load ~/Library/LaunchAgents/com.example.gotray.plist`.
4. The tray runs within the logged-in session. Configuration is stored at `~/Library/Application Support/GoTray/config.enc` by default.

## 5. Development sandbox

Use `scripts/install_dev.sh` to install a development copy alongside production. The script sets up an isolated environment file (`/etc/gotray/dev-<username>.env`), builds the binary into `/opt/gotray-dev`, and creates a dedicated config path (`/var/lib/gotray-dev/<username>/config.enc`). Enable it with `systemctl enable --now gotray-dev@<username>.service` so it does not interfere with production data.

## 6. Monitoring and troubleshooting

* `journalctl -u gotray@<username>.service` (Linux) streams tray logs for the specified user.
* Task Scheduler history (Windows) or Console.app (macOS) shows launch failures.
* Regenerate the configuration by deleting the encrypted file—GoTray recreates the defaults on the next start.
* Always protect `GOTRAY_SECRET`. Rotate it and re-encrypt the configuration if compromise is suspected.

Refer back to [README.md](../README.md#command-line-management) for CLI management details.
