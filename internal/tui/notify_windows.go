//go:build windows

package tui

import (
	"github.com/gen2brain/beeep"
)

func sendDesktopNotification(title, body string) error {
	return beeep.Notify(title, body, "")
}
