package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/atotto/clipboard"
)

// UTF emoji
const (
	flagNL  = "\U0001F1F3\U0001F1F1" //Netherlands
	flagFR  = "\U0001F1EB\U0001F1E7" //France
	flagUG  = "\U0001F1FA\U0001F1EC" //Uganda
	pingOK  = "\U00002705"
	pingERR = "\U0000274C"
)

// Wait  - python-like thread.Wait
type Wait struct {
	lock   sync.RWMutex
	_isSet bool
}

// IfSet set true and return true if previous status is false
func (w *Wait) IfSet() (result bool) {
	w.lock.Lock()
	defer w.lock.Unlock()
	result = !w._isSet
	w._isSet = true
	return
}

// Clear set false
func (w *Wait) Clear() {
	w.lock.Lock()
	defer w.lock.Unlock()
	w._isSet = false
}

// IsSet return current status
func (w *Wait) IsSet() bool {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w._isSet
}

// Set true
func (w *Wait) Set() {
	w.lock.Lock()
	defer w.lock.Unlock()
	w._isSet = true
}

func printErr(format string, a ...interface{}) {
	format += "\n"
	fmt.Fprintf(os.Stderr, format, a...)
}

func sReplaceAll(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

func fillMask(mask string, data *serverInfo) string {
	mask = sReplaceAll(mask, "{ID}", data.ID)
	mask = sReplaceAll(mask, "{NAME}", data.NAME)
	mask = sReplaceAll(mask, "{IPv4}", data.IPv4)
	mask = sReplaceAll(mask, "{IPv6}", data.IPv6)
	mask = sReplaceAll(mask, "{STATE}", data.STATE)
	mask = sReplaceAll(mask, "{REGION}", data.REGION)
	mask = sReplaceAll(mask, "{PING}", data.pingMS)
	if data.isIPv4 {
		mask = sReplaceAll(mask, "{IPvX}", data.IPv4)
	} else if data.isIPv6 {
		mask = sReplaceAll(mask, "{IPvX}", data.IPv6)
	} else {
		mask = sReplaceAll(mask, "{IPvX}", "IPvX")
	}
	return mask
}

func fillView(mask string, data *serverInfo) string {
	mask = fillMask(mask, data)
	switch data.REGION {
	case "par1":
		mask = sReplaceAll(mask, "{FLAG}", flagFR)
	case "ams1":
		mask = sReplaceAll(mask, "{FLAG}", flagNL)
	default:
		mask = sReplaceAll(mask, "{FLAG}", flagUG)
	}
	if data.pingState {
		mask = sReplaceAll(mask, "{ALIVE}", pingOK)
	} else {
		mask = sReplaceAll(mask, "{ALIVE}", pingERR)
	}
	return mask
}

func writeToClipboard(idx int, cfg *settingsStorage, srv *serversInfo) (err error) {
	cfg.L.RLock()
	mask := cfg.D.CopyMask
	cfg.L.RUnlock()

	srv.L.RLock()
	defer srv.L.RUnlock()

	count := len(srv.ServersList)
	if idx >= count || idx < 0 {
		err = fmt.Errorf("Wrong menu index: %d", idx)
	} else {
		id := srv.ServersList[idx]
		if item, ok := srv.D[id]; ok {
			err = clipboard.WriteAll(fillMask(mask, item))
		} else {
			panic(fmt.Errorf("serversInfo: Corrupted"))
		}
	}
	return
}
