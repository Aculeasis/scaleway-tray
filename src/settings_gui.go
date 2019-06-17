package main

import (
	"fmt"
	"runtime"

	"github.com/andlabs/ui"
	"github.com/getlantern/systray"
)

// GUI ...
type GUI struct {
	config *settingsStorage
	wait   Wait
	// call when "Quit" clicked
	quitCallback     func()
	scalewayCallback func(cfgActionID)
	pingCallback     func()
	// unsafe
	_setter func()
}

func makeGUI(config *settingsStorage, scalewayCallback func(cfgActionID), pingCallback func(), quitCallback func()) *GUI {
	g := GUI{}
	g.config = config
	g.scalewayCallback = scalewayCallback
	g.pingCallback = pingCallback
	g.quitCallback = quitCallback
	g.wait.Set()
	return &g
}

// Show Make and show gui.
func (g *GUI) Show() {
	if g.wait.IfSet() {
		ui.QueueMain(g.showGUI)
	}
}

// Destroy gui
func (g *GUI) Destroy() {
	ui.QueueMain(ui.Quit)
}

var mainwin *ui.Window

// Set gui values from settingsData
func (g *GUI) callSetter() {
	if g._setter != nil {
		g._setter()
	}
}

// Call where mainwin is destroy - unlink all gui method and mark gui as "Closed"
func (g *GUI) clearALL() {
	g._setter = nil
	g.wait.Clear()
}

//Start thread
func (g *GUI) Start() {
	go g.loop()
}

// GUI loop. Run on goroutine
func (g *GUI) loop() {
	// WARNING: If systray.Quit() call before ui.Quit finished - systray.Run never be stopped (Linux bug)
	defer systray.Quit()
	ui.OnShouldQuit(func() bool {
		return true
	})
	// mark gui as "Closed"
	err := ui.Main(g.wait.Clear)
	if err != nil {
		panic(fmt.Errorf("UI error %e", err))
	}
	g.clearALL()

}

// Make and Show gui
func (g *GUI) showGUI() {
	mainwin = ui.NewWindow(appName, 64, 48, true)
	mainwin.SetMargined(true)
	mainwin.OnClosing(func(*ui.Window) bool {
		g.clearALL()
		return true
	})
	box := ui.NewVerticalBox()
	box.SetPadded(true)
	tab := ui.NewTab()
	mainwin.SetChild(box)
	mainwin.SetMargined(true)
	box.Append(tab, false)

	tab.Append("Settings", g.makeTabSettings())
	tab.SetMargined(0, true)

	tab.Append("Info", g.makeInfoSettings())
	tab.SetMargined(1, true)

	box.Append(g.makeButtonsSettings(), true)

	mainwin.Show()
}

func (g *GUI) makeTabSettings() ui.Control {
	vbox := ui.NewVerticalBox()
	form := ui.NewForm()
	vbox.SetPadded(true)
	form.SetPadded(true)
	vbox.Append(form, true)

	elOrganizationID := ui.NewEntry()
	elAccessKey := ui.NewEntry()
	elSecretKey := ui.NewPasswordEntry()
	form.Append("Organization", elOrganizationID, false)
	form.Append("Access Key", elAccessKey, false)
	form.Append("Secret Key", elSecretKey, false)
	form.Append("", ui.NewLabel(""), false)

	elMenuMask := ui.NewEntry()
	elCopyMask := ui.NewEntry()
	form.Append("Menu format", elMenuMask, false)
	form.Append("Copy format", elCopyMask, false)
	form.Append("", ui.NewLabel(""), false)

	elCheckInterval := ui.NewSpinbox(0, 3600*24*30)
	elPingInterval := ui.NewSpinbox(0, 3600*24*30)
	form.Append("Check interval", elCheckInterval, false)
	form.Append("Ping interval", elPingInterval, false)

	g._setter = func() {
		g.config.L.RLock()
		defer g.config.L.RUnlock()

		elOrganizationID.SetText(g.config.D.OrganizationID)
		elAccessKey.SetText(g.config.D.AccessKey)
		elSecretKey.SetText(g.config.D.SecretKey)

		elMenuMask.SetText(g.config.D.ViewMask)
		elCopyMask.SetText(g.config.D.CopyMask)

		elCheckInterval.SetValue(g.config.D.CheckInterval)
		elPingInterval.SetValue(g.config.D.PingInterval)
	}

	elOrganizationID.OnChanged(func(*ui.Entry) {
		g.config.L.Lock()
		defer g.config.L.Unlock()
		g.config.D.OrganizationID = elOrganizationID.Text()
		g.scalewayCallback(scalewayCFGSignal)
	})
	elAccessKey.OnChanged(func(*ui.Entry) {
		g.config.L.Lock()
		defer g.config.L.Unlock()
		g.config.D.AccessKey = elAccessKey.Text()
		g.scalewayCallback(scalewayCFGSignal)
	})
	elSecretKey.OnChanged(func(*ui.Entry) {
		g.config.L.Lock()
		defer g.config.L.Unlock()
		g.config.D.SecretKey = elSecretKey.Text()
		g.scalewayCallback(scalewayCFGSignal)
	})

	elMenuMask.OnChanged(func(*ui.Entry) {
		g.config.L.Lock()
		defer g.config.L.Unlock()
		g.config.D.ViewMask = elMenuMask.Text()
		g.scalewayCallback(scalewayMaskSignal)
	})
	elCopyMask.OnChanged(func(*ui.Entry) {
		g.config.L.Lock()
		defer g.config.L.Unlock()
		g.config.D.CopyMask = elCopyMask.Text()
	})

	elCheckInterval.OnChanged(func(*ui.Spinbox) {
		g.config.L.Lock()
		defer g.config.L.Unlock()
		g.config.D.CheckInterval = elCheckInterval.Value()
		g.scalewayCallback(scalewayCFGSignal)
	})
	elPingInterval.OnChanged(func(*ui.Spinbox) {
		g.config.L.Lock()
		defer g.config.L.Unlock()
		g.config.D.PingInterval = elPingInterval.Value()
		g.pingCallback()
	})

	g.callSetter()
	return vbox
}

func (g *GUI) makeButtonsSettings() ui.Control {
	vbox := ui.NewVerticalBox()
	grid := ui.NewGrid()
	vbox.Append(grid, false)
	vButtons := ui.NewHorizontalBox()
	vButtons.SetPadded(true)
	grid.Append(vButtons, 0, 0, 1, 1, true, ui.AlignCenter, true, ui.AlignFill)

	resetButton := ui.NewButton("Reset")
	exportButton := ui.NewButton("Export")
	importButton := ui.NewButton("Import")
	quitButton := ui.NewButton("Quit")

	vButtons.Append(resetButton, false)
	vButtons.Append(importButton, false)
	vButtons.Append(exportButton, false)
	vButtons.Append(quitButton, false)

	resetButton.OnClicked(func(*ui.Button) {
		g.config.ResetSettings()
		g.callSetter()
		g.scalewayCallback(scalewayUpdateSignal)
		g.pingCallback()
		ui.MsgBox(mainwin, "Reset", "OK.")
	})
	importButton.OnClicked(func(*ui.Button) {
		if path := ui.OpenFile(mainwin); path != "" {
			if err := g.config.LoadFrom(path); err != nil {
				ui.MsgBoxError(mainwin, "LOAD ERROR", fmt.Sprintf("%v", err))
			} else {
				g.callSetter()
				g.scalewayCallback(scalewayUpdateSignal)
				g.pingCallback()
				ui.MsgBox(mainwin, "Load", "OK.")
			}
		}
	})
	exportButton.OnClicked(func(*ui.Button) {
		if path := ui.SaveFile(mainwin); path != "" {
			if err := g.config.SaveTo(path); err != nil {
				ui.MsgBoxError(mainwin, "SAVE ERROR", fmt.Sprintf("%v", err))
			} else {
				ui.MsgBox(mainwin, "Save", "OK.")
			}
		}
	})
	quitButton.OnClicked(func(*ui.Button) {
		g.quitCallback()
	})
	return vbox
}

func (g *GUI) makeInfoSettings() ui.Control {
	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)
	entry := ui.NewForm()
	entry.SetPadded(true)
	vbox.Append(entry, true)
	label := func(key, value string) {
		entry.Append(key, ui.NewLabel(value), false)
	}
	label("Version", Version)
	label("Commit", GitCommit)
	label("Build date", BuildDate)
	label("OS", runtime.GOOS+", "+runtime.GOARCH)
	label("Build", runtime.Compiler+","+runtime.Version())

	return vbox
}
