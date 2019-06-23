package main

import (
	"os"
	"os/signal"
	"syscall"
)

type signalHandler struct {
	chain chan os.Signal
}

func newSignalHandler() *signalHandler {
	sh := signalHandler{}
	sh.chain = make(chan os.Signal, 5)
	signal.Notify(sh.chain, syscall.SIGINT, syscall.SIGTERM)
	return &sh
}

// Send signal
func (sh *signalHandler) Send() {
	select {
	case sh.chain <- syscall.SIGTERM:
	default:
	}
}

func (sh *signalHandler) Start(callbacks ...func()) {
	go sh.goSignalHandler(callbacks...)
}

func (sh *signalHandler) goSignalHandler(callbacks ...func()) {
	_ = <-sh.chain
	for _, callback := range callbacks {
		callback()
	}
}
