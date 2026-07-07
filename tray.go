package main

import (
	_ "embed"
	"fmt"
	"log"
	"runtime"

	"fyne.io/systray"
)

//go:embed assets/icon.ico
var iconICO []byte

//go:embed assets/tray.png
var trayPNG []byte

//go:embed assets/tray-template.png
var trayTemplatePNG []byte

// runTray blocks until the user quits from the tray menu, then calls onExit.
func runTray(port int, onExit func()) {
	systray.Run(func() {
		switch runtime.GOOS {
		case "windows":
			systray.SetIcon(iconICO)
		case "darwin":
			// Template icons render correctly in both light and dark menu bars.
			systray.SetTemplateIcon(trayTemplatePNG, trayTemplatePNG)
		default:
			systray.SetIcon(trayPNG)
		}
		systray.SetTooltip(fmt.Sprintf("Folder Opener %s — 127.0.0.1:%d", version, port))

		title := systray.AddMenuItem(fmt.Sprintf("Folder Opener %s", version), "")
		title.Disable()
		addr := systray.AddMenuItem(fmt.Sprintf("Listening on 127.0.0.1:%d", port), "")
		addr.Disable()
		systray.AddSeparator()

		autostartItem := systray.AddMenuItemCheckbox(
			"Start at Login", "Launch Folder Opener automatically when you log in", autostartEnabled())
		systray.AddSeparator()
		quit := systray.AddMenuItem("Quit", "Stop the Folder Opener server")

		go func() {
			for {
				select {
				case <-autostartItem.ClickedCh:
					if autostartItem.Checked() {
						if err := disableAutostart(); err != nil {
							log.Printf("disable autostart: %v", err)
						} else {
							autostartItem.Uncheck()
						}
					} else {
						if err := enableAutostart(); err != nil {
							log.Printf("enable autostart: %v", err)
						} else {
							autostartItem.Check()
						}
					}
				case <-quit.ClickedCh:
					systray.Quit()
				}
			}
		}()
	}, onExit)
}
