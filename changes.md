# Change Log

- 2025-10-21T21:57:20Z - Feature - Initial GoTray implementation with encrypted configuration, system tray interface, and deployment scripts.
- 2025-10-21T23:19:36Z - Feature - Added GitHub Actions workflow to compile cross-platform executables.
- 2025-10-21T23:31:59Z - Fix - Added build-tagged tray stub to enable non-Windows builds without cgo.
- 2025-10-22T00:14:48Z - Fix - Embedded GOTRAY_SECRET through GitHub Actions secrets with local development fallback guidance.
