//go:generate goversioninfo -icon=../icon/icon.ico -manifest=../goversioninfo/goversioninfo.exe.manifest -64 -platform-specific=false

package main

import (
	"github.com/getlantern/systray"
)

const appName = "scaleway-tray"

// BuildDate override in building
var BuildDate string

// GitCommit ...
var GitCommit string

// Version ...
var Version = "0.0.0"

func main() {
	systray.Run(onReady, nil)
}

func onReady() {
	defer systray.Quit()
	stopChan := make(chan struct{}, 1)
	stopMe := func() {
		select {
		case stopChan <- struct{}{}:
		default:
		}
	}
	menu := newMenuPool(20)
	systray.AddSeparator()
	mSettings := systray.AddMenuItem("Settings", "Settings")
	mQuit := systray.AddMenuItem("Quit", "Quit")

	stopper := newSignalHandler()
	settings := newSettingsStorage()
	scaleway := newScalewayWorker(settings, menu)
	pinger := newPingWorker(settings, scaleway.servers, scaleway.CFGChange)
	gui := newSettingsGUI(settings, scaleway.CFGChange, pinger.CFGChange, stopper.Send)

	systray.SetIcon(iconData)
	systray.SetTitle("Scaleway Tray")
	systray.SetTooltip("Scaleway Tray")

	stopper.Start(pinger.Quit, scaleway.Quit, gui.Quit, stopMe)
	scaleway.Start(pinger.PingSignal)
	gui.Start()
	pinger.Start()
	for {
		select {
		case <-mQuit.ClickedCh:
			stopper.Send()
		case <-mSettings.ClickedCh:
			gui.Show()
		case <-stopChan:
			if err := settings.Save(); err != nil {
				printErr("settings.save: %v", err)
			}
			// WARNING: If systray.Quit() call before ui.Quit finished - systray.Run never be stopped (in Linux)
			gui.Wait()
			return
		case num := <-menu.WaitSignal():
			if err := writeToClipboard(num, settings, scaleway.servers); err != nil {
				printErr("WriteToClipboard: %v", err)
			}
		}
	}
}
