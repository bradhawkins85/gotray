//go:build windows

package main

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
)

func init() {
	if shouldShowConsole(os.Args[1:]) {
		return
	}
	hideConsoleWindow()
}

func shouldShowConsole(args []string) bool {
	if os.Getenv("GOTRAY_SHOW_CONSOLE") != "" {
		return true
	}

	for _, raw := range args {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		normalized := strings.ToLower(strings.TrimLeft(trimmed, "-/"))
		switch {
		case normalized == "console":
			return true
		case strings.HasPrefix(normalized, "console="):
			value := strings.TrimPrefix(normalized, "console=")
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				continue
			}
			if parsed {
				return true
			}
		}
	}

	return false
}

func hideConsoleWindow() {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	user32 := windows.NewLazySystemDLL("user32.dll")

	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	showWindow := user32.NewProc("ShowWindow")
	freeConsole := kernel32.NewProc("FreeConsole")

	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd == 0 {
		return
	}

	const swHide = 0
	showWindow.Call(hwnd, swHide)
	freeConsole.Call()
}
