//go:build darwin && cgo
// +build darwin,cgo

package menu

import "github.com/getlantern/systray"

func setTemplateIcon(icon []byte) {
	systray.SetTemplateIcon(icon, icon)
}
