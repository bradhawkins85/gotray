# GoTray

GoTray is a cross-platform system tray helper written in Go. It encrypts its configuration on disk and can be driven entirely from the command line for scripted updates.

## Prerequisites

* Go 1.21 or newer
* A strong passphrase exported as the `GOTRAY_SECRET` environment variable. This secret is required every time the application starts because it encrypts and decrypts the menu configuration on disk.

Optional environment variable:

* `GOTRAY_CONFIG_PATH` – overrides the default configuration location. By default the encrypted file is stored in `~/.config/gotray/config.enc` (respecting your operating system's user configuration directory).

You can copy `.env.example` and adjust it to suit your environment:

```bash
cp .env.example .env
```

## Building and running the tray

```bash
export GOTRAY_SECRET="your-strong-passphrase"
go run ./cmd/gotray
```

When the tray starts for the first time it seeds the configuration with a set of defaults. Any subsequent changes are encrypted with the secret you supplied.

## Command-line management

GoTray ships with a CLI that lets you manage menu items without opening a graphical interface. Every command must be prefixed with the desired verb (`add`, `update`, `delete`, or `list`) followed by its switches. Flags accept either `--` or `-` prefixes as well as `/` prefixes on Windows.

### Adding items

```
go run ./cmd/gotray add --type text --label "Welcome"
```

Supported `--type` values are:

* `text` – renders a plain text entry.
* `divider` – inserts a separator.
* `command` – launches an executable.
* `url` – opens the provided link in the default browser.

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

The `list` command prints the currently configured entries sorted by their creation timestamp.

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

### Exit codes and errors

All commands return a non-zero exit code on error and print a helpful message describing what went wrong (for example, missing required flags or an unknown identifier). This makes it safe to script changes in provisioning tools.

## Configuration storage

* Configurations are encrypted with AES-GCM using a key derived from your `GOTRAY_SECRET` via scrypt.
* The file is replaced atomically on save to protect against partial writes.
* Timestamps are stored in UTC and include both creation and last-updated times.

## Troubleshooting

* **"GOTRAY_SECRET environment variable is required"** – ensure the variable is exported before running any command.
* **"unknown command" errors** – verify that you spelled the verb correctly (`add`, `update`, `delete`, `list`).
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
