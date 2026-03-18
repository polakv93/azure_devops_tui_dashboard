//go:build !windows

package tui

func sendDesktopNotification(title, body string) error {
	return nil
}
