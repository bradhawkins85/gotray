package menu

import "net/url"

// openURL validates the provided URL before deferring to the platform-specific
// launcher. Validation is shared so the OS-specific launchers can remain
// minimal and avoid additional dependencies.
func openURL(raw string) {
	if raw == "" {
		return
	}
	if _, err := url.ParseRequestURI(raw); err != nil {
		return
	}

	launchURL(raw)
}
