//go:generate goversioninfo -icon=../icon/icon.ico -manifest=../goversioninfo/goversioninfo.exe.manifest -64 -platform-specific=false

package main

import (
	"strconv"
	"sync"

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
	wg := sync.WaitGroup{}
	wg.Add(1)
	systray.Run(func() {
		onReady(&wg)
	}, nil)
	wg.Wait()
}

func makeMenuPool(size int) (*[]*systray.MenuItem, <-chan int) {
	pool := make([]*systray.MenuItem, size)
	channel := make(chan int, 1)
	for idx := range pool {
		pool[idx] = systray.AddMenuItem(strconv.Itoa(idx), "")
		pool[idx].Hide()
		go func(id int, ch chan struct{}) {
			for range ch {
				channel <- id
			}
		}(idx, pool[idx].ClickedCh)
	}
	return &pool, channel
}

func onReady(wg *sync.WaitGroup) {
	defer wg.Done()
	stopChan := make(chan struct{}, 1)
	stopMe := func() {
		select {
		case stopChan <- struct{}{}:
		default:
		}
	}
	menuPool, menuChannel := makeMenuPool(20)
	systray.AddSeparator()
	mSettings := systray.AddMenuItem("Settings", "Settings")
	mQuit := systray.AddMenuItem("Quit", "Quit")

	stopper := makeSignalHandler()
	settings := makeSettingsStorage()
	scaleway := makeScalewayWorker(settings, menuPool)
	pinger := makePingWorker(settings, scaleway.servers, scaleway.CFGChange)
	gui := makeGUI(settings, scaleway.CFGChange, pinger.CFGChange, stopper.Send)

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
			return
		case num := <-menuChannel:
			if err := scaleway.WriteToClipboard(num); err != nil {
				printErr("WriteToClipboard: %v", err)
			}
		}
	}
}
