# Change Log

- 2025-10-22T01:22:45Z - Fix - Resolved tray runner build conflicts by consolidating platform tray management logic and restoring cross-platform compilation.
- 2025-10-22T00:53:38Z - Feature - Reworked GoTray into a system-context service with authenticated per-session tray agents, cross-platform service documentation, and updated installers.
- 2025-10-22T00:43:27Z - Feature - Added CLI support for deleting all menu items with a single command.
- 2025-10-22T01:02:50Z - Fix - Added legacy build tags to tray implementations to restore compatibility with older Go toolchains.
- 2025-10-22T00:37:31Z - Feature - Introduced positional menu management, including label-based deletion, move support, and default order sequencing.
- 2025-10-22T00:19:37Z - Feature - Documented command-line usage and examples in README.
- 2025-10-21T21:57:20Z - Feature - Initial GoTray implementation with encrypted configuration, system tray interface, and deployment scripts.
- 2025-10-21T23:19:36Z - Feature - Added GitHub Actions workflow to compile cross-platform executables.
- 2025-10-21T23:31:59Z - Fix - Added build-tagged tray stub to enable non-Windows builds without cgo.
- 2025-10-22T00:14:48Z - Fix - Embedded GOTRAY_SECRET through GitHub Actions secrets with local development fallback guidance.
