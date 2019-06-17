package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

func printErr(format string, a ...interface{}) {
	format += "\n"
	fmt.Fprintf(os.Stderr, format, a...)
}

func sReplaceAll(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

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
