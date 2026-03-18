//go:build windows

package tui

import (
	"sync"

	"github.com/gen2brain/beeep"
)

var notificationAppNameOnce sync.Once

func sendDesktopNotification(title, body string) error {
	notificationAppNameOnce.Do(func() {
		beeep.AppName = "azdo-tui"
	})

	return beeep.Notify(title, body, "")
}
