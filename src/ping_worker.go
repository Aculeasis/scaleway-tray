package main

import (
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/sparrc/go-ping"
)

type pingWorker struct {
	servers       *serversInfo
	data          *settingsStorage
	stopChan      chan os.Signal
	cfgChangeChan chan struct{}
	pingSignals   chan struct{}
	scalewayCFG   func(cfgActionID)
}

func newPingWorker(data *settingsStorage, servers *serversInfo, scalewayCFG func(cfgActionID)) *pingWorker {
	pg := pingWorker{}

	pg.stopChan = make(chan os.Signal, 1)
	pg.cfgChangeChan = make(chan struct{}, 1)
	pg.pingSignals = make(chan struct{}, 1)

	pg.data = data
	pg.servers = servers
	pg.scalewayCFG = scalewayCFG

	return &pg
}

func (pg *pingWorker) Start() {
	go pg.loop()
}

func (pg *pingWorker) loop() {
	var timerChan <-chan time.Time
	var pingInterval time.Duration
	initTimer := func() {
		pg.data.L.RLock()
		defer pg.data.L.RUnlock()
		pingInterval = time.Duration(pg.data.D.PingInterval)
	}

	makeTimer := func() {
		if pingInterval >= 1 {
			timerChan = time.After(time.Second * pingInterval)
		} else {
			timerChan = make(<-chan time.Time, 1)
		}
	}

	initTimer()
	makeTimer()

	for {
		select {
		case <-pg.stopChan:
			return
		case <-time.After(time.Microsecond * 800):
			select {
			case <-pg.cfgChangeChan:
				initTimer()
				makeTimer()
			default:
			}
		case <-pg.pingSignals:
			if pingInterval >= 1 {
				pg.ping()
				makeTimer()
			}
		case <-timerChan:
			pg.ping()
			makeTimer()
		}
	}
}

func (pg *pingWorker) ping() {
	wg := sync.WaitGroup{}

	pg.servers.L.RLock()
	for id, item := range pg.servers.D {
		host := item.IPv4
		if !item.isIPv4 {
			host = item.IPv6
		}
		if item.isIPv4 || item.isIPv6 {
			wg.Add(1)
			go pg.pingHost(host, id, item.pingState, item.pingMS, &wg)
		}
	}
	pg.servers.L.RUnlock()

	wg.Wait()
}

func (pg *pingWorker) pingHost(host string, id serverID, oldState bool, oldPingMS string, wg *sync.WaitGroup) {
	defer wg.Done()
	pinger, err := ping.NewPinger(host)
	if err != nil {
		printErr("NewPinger %s: %v", host, err)
		return
	}
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}
	pinger.Count = 1
	pinger.Timeout = time.Second * 5
	pinger.Run()
	statistics := pinger.Statistics()
	newState := statistics.PacketsRecv > 0
	newPingMS := strconv.FormatInt((statistics.AvgRtt / time.Millisecond).Nanoseconds(), 10)
	if oldState == newState && (newPingMS == oldPingMS || !newState) {
		return
	}
	pg.servers.L.Lock()
	defer pg.servers.L.Unlock()
	if item, ok := pg.servers.D[id]; ok {
		item.pingState = newState
		if newState {
			item.pingMS = newPingMS
		}
		pg.scalewayCFG(scalewayDrawSignal)
	}
}

func (pg *pingWorker) Quit() {
	select {
	case pg.stopChan <- syscall.SIGTERM:
	default:
	}
}

func (pg *pingWorker) CFGChange() {
	select {
	case pg.cfgChangeChan <- struct{}{}:
	default:
	}
}

func (pg *pingWorker) PingSignal() {
	select {
	case pg.pingSignals <- struct{}{}:
	default:
	}
}
