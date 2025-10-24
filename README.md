# GoTray

GoTray is a cross-platform system tray helper written in Go. It encrypts its configuration on disk, runs entirely in a user session, and can be driven from the command line for scripted updates. Version 3 simplifies the deployment model so every desktop user launches a stand-alone instance without any background service or IPC layer.

## Prerequisites

* Go 1.21 or newer
* A strong passphrase exported as the `GOTRAY_SECRET` environment variable when running the tray or CLI commands. This secret encrypts and decrypts the menu configuration on disk.

Optional environment variables:

* `GOTRAY_CONFIG_PATH` – overrides the default configuration location. By default the encrypted file is stored in `~/.config/gotray/config.enc` (respecting your operating system's user configuration directory).
* `GOTRAY_RUN_MODE` – set the default sub-command when no CLI arguments are supplied. Defaults to `run` so the binary behaves as a long-running tray application for the invoking user.

You can copy `.env.example` and adjust it to suit your environment:

```bash
cp .env.example .env
```

## Running the stand-alone tray application

Build the binary and launch it within the target desktop session:

```bash
export GOTRAY_SECRET="your-strong-passphrase"
go run ./cmd/gotray run
```

When the tray starts for the first time it seeds the configuration with a set of defaults and encrypts them using the supplied secret. Subsequent edits are written straight to the encrypted configuration file for that user.

Detailed platform-specific installation steps for Linux, macOS, and Windows live in [docs/user-setup.md](docs/user-setup.md).

## Command-line management

GoTray ships with a CLI that lets you manage menu items without opening a graphical interface. Execute the commands on the same machine that owns the encrypted configuration file to edit the menu without interrupting the running tray instance. Every command must be prefixed with the desired verb (`add`, `update`, `delete`, `list`, `move`, `export`, or `import`) followed by its switches. Flags accept either `--` or `-` prefixes as well as `/` prefixes on Windows.

### Adding items

```
go run ./cmd/gotray add --type text --label "Welcome"
```

Supported `--type` values are:

* `text` – renders a plain text entry.
* `divider` – inserts a separator.
* `command` – launches an executable.
* `url` – opens the provided link in the default browser.
* `menu` – creates a submenu container that can hold nested entries.
* `quit` – closes the GoTray application when selected.

Additional switches for `add`:

| Flag | Applies to | Description |
| ---- | ---------- | ----------- |
| `--label` | `text`, `command`, `url` | Display label shown in the tray. Required for these types. |
| `--description` | all | Optional tooltip text. |
| `--command` | `command` | Executable or script to run. Required for command items. |
| `--args` | `command` | Comma-separated list of arguments passed to the executable. |
| `--workdir` | `command` | Working directory for the process. |
| `--url` | `url` | Destination URL opened by the system browser. Required for URL items. |

Example: add a command menu item that launches a log viewer.

```
go run ./cmd/gotray add \
  --type command \
  --label "Tail logs" \
  --command /usr/bin/tail \
  --args "/var/log/system.log,-f" \
  --workdir /usr/bin \
  --description "Follow the system log output"
```

### Listing items

The `list` command prints the currently configured entries in their display order.

```
go run ./cmd/gotray list
```

Example output:

```
ID                                     Type     Label                Updated (UTC)
8c5f0fd3-1e38-43f6-a29d-1d6bb1895fae   text     Welcome              2024-04-11T09:22:18Z
c33ad357-0c0e-4efa-9e15-6f6cfb04f36b   command  Tail logs            2024-04-11T09:23:52Z
```

### Updating items

To update an item you must supply its `--id`, which you can obtain from the `list` command. Only the flags you provide are changed; omitted flags keep their existing values.

```
go run ./cmd/gotray update \
  --id 8c5f0fd3-1e38-43f6-a29d-1d6bb1895fae \
  --label "Welcome aboard" \
  --description "Greeting shown at the top of the menu"
```

You can also change the type of an entry:

```
go run ./cmd/gotray update \
  --id c33ad357-0c0e-4efa-9e15-6f6cfb04f36b \
  --type url \
  --url https://status.example.com
```

When switching to a new type, remember to include any required flags for the target type (`--url` for `url`, `--command` for `command`, etc.).

### Deleting items

Remove an entry by its identifier:

```
go run ./cmd/gotray delete --id c33ad357-0c0e-4efa-9e15-6f6cfb04f36b
```

To clear the entire menu and start fresh, delete all items in one command:

```
go run ./cmd/gotray delete --all
```

### Exporting and importing menus

Back up or move the entire menu structure with `export` and `import`. The configuration is serialized to JSON and wrapped in a Base64 string so it can be pasted safely into scripts or secrets managers.

Export the current menu to stdout:

```
go run ./cmd/gotray export
```

Import a previously exported payload (replacing the current menu items):

```
go run ./cmd/gotray import --data "$(go run ./cmd/gotray export)"
```

You can also read the payload from a file:

```
go run ./cmd/gotray import --file backup.txt
```

During import the CLI validates every menu item and ensures parent-child relationships remain intact before encrypting the configuration again.

### Exit codes and errors

All commands return a non-zero exit code on error and print a helpful message describing what went wrong (for example, missing required flags or an unknown identifier). This makes it safe to script changes in provisioning tools.

## Configuration storage

* Configurations are encrypted with AES-GCM using a key derived from your `GOTRAY_SECRET` via scrypt.
* The file is replaced atomically on save to protect against partial writes.
* Timestamps are stored in UTC and include both creation and last-updated times.

## Troubleshooting

* **"GOTRAY_SECRET environment variable is required"** – ensure the variable is exported before running the tray or CLI commands.
* **"unknown command" errors** – verify that you spelled the verb correctly (`add`, `update`, `delete`, `list`, `move`, `export`, `import`).
* **"item with id ... not found"** – use `go run ./cmd/gotray list` to confirm the identifier before updating or deleting.

## Development

Run the test suite:

```bash
go test ./...
```

Lint and vet the codebase:

```bash
go vet ./...
```

Pull requests should include documentation and changelog updates whenever you introduce new functionality.
