package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/utils"

	"github.com/atotto/clipboard"
	"github.com/getlantern/systray"
)

// UTF emoji
const (
	flagNL  = "\U0001F1F3\U0001F1F1" //Netherlands
	flagFR  = "\U0001F1EB\U0001F1E7" //France
	flagUG  = "\U0001F1FA\U0001F1EC" //Uganda
	pingOK  = "\U00002705"
	pingERR = "\U0000274C"
)

type cfgActionID uint8

const (
	scalewayUpdateSignal cfgActionID = iota
	scalewayCFGSignal
	scalewayMaskSignal
	scalewayDrawSignal
)

type serverID string
type serverInfo struct {
	ID        string
	NAME      string
	IPv4      string
	IPv6      string
	STATE     string
	REGION    string
	isIPv4    bool
	isIPv6    bool
	pingState bool
	pingMS    string
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

type serversInfo struct {
	D map[serverID]*serverInfo
	L sync.RWMutex
	// for ID position saving
	ServersList []serverID
}

type scalewayWorker struct {
	servers     *serversInfo
	config      *settingsStorage
	menu        *[]*systray.MenuItem
	stopChan    chan os.Signal
	signalsChan chan cfgActionID
}

func makeScalewayWorker(config *settingsStorage, menu *[]*systray.MenuItem) *scalewayWorker {
	sw := scalewayWorker{}
	sw.servers = &serversInfo{D: map[serverID]*serverInfo{}, ServersList: []serverID{}}
	sw.stopChan = make(chan os.Signal, 1)
	sw.signalsChan = make(chan cfgActionID, 3)

	sw.config = config
	sw.menu = menu

	return &sw
}

func (sw *scalewayWorker) Start(firstRunCallback func()) {
	go sw.loop(firstRunCallback)
}

func (sw *scalewayWorker) loop(firstRunCallback func()) {
	var timerChan <-chan time.Time
	var oldmask string
	var firstRun bool

	var updateInterval time.Duration
	var enable bool
	initTimer := func() {
		sw.config.L.RLock()
		defer sw.config.L.RUnlock()
		updateInterval = time.Duration(sw.config.D.CheckInterval)
		enable = sw.config.D.OrganizationID != "" &&
			sw.config.D.AccessKey != "" &&
			sw.config.D.SecretKey != ""
	}
	makeTimer := func() {
		if updateInterval >= 10 && enable {
			timerChan = time.After(time.Second * updateInterval)
			firstRun = true
		} else {
			timerChan = make(<-chan time.Time, 1)
		}
	}
	maskChange := func() {
		mask := sw.getViewMask()
		if mask != oldmask {
			oldmask = mask
			sw.updateMenu(oldmask, false)
		}
	}

	initTimer()
	makeTimer()
	if firstRun {
		sw.updateScaleway()
		makeTimer()
		if firstRunCallback != nil {
			firstRunCallback()
		}
	}

	for {
		select {
		case <-sw.stopChan:
			return
		case <-time.After(time.Second * 1):
			select {
			case id := <-sw.signalsChan:
				if id == scalewayDrawSignal {
					sw.updateMenu(sw.getViewMask(), false)
					break
				}
				if id == scalewayCFGSignal || id == scalewayUpdateSignal {
					initTimer()
					makeTimer()
				}
				if id == scalewayMaskSignal || id == scalewayUpdateSignal {
					maskChange()
				}
			default:
			}
		case <-timerChan:
			sw.updateScaleway()
			makeTimer()
		}
	}
}

func (sw *scalewayWorker) updateMenu(mask string, menuChange bool) {
	if menuChange {
		for _, item := range *sw.menu {
			item.Hide()
		}
	}
	sw.servers.L.RLock()
	defer sw.servers.L.RUnlock()
	size := len(sw.servers.ServersList)
	mSize := len(*sw.menu)
	if size > mSize {
		size = mSize
	}
	for idx := 0; idx < size; idx++ {
		id := sw.servers.ServersList[idx]
		if item, ok := sw.servers.D[id]; ok {
			(*sw.menu)[idx].SetTitle(fillView(mask, item))
			if menuChange {
				(*sw.menu)[idx].Show()
			}
		} else {
			panic(fmt.Errorf(""))
		}
	}

}

func (sw *scalewayWorker) updateScaleway() {
	sw.config.L.RLock()
	client, err := scw.NewClient(
		// Get your credentials at https://console.scaleway.com/account/credentials
		scw.WithDefaultProjectID(sw.config.D.OrganizationID),
		scw.WithAuth(sw.config.D.AccessKey, sw.config.D.SecretKey),
	)
	sw.config.L.RUnlock()
	if err != nil {
		printErr("NewClient: %v", err)
		return
	}
	// Create SDK objects for Scaleway Instance product
	instanceAPI := instance.NewAPI(client)
	// Call the ListServers method on the Instance SDK
	all := &instance.ListServersResponse{}
	var zones = []utils.Zone{utils.ZoneFrPar1, utils.ZoneNlAms1} //utils.AllZones
	for _, zone := range zones {
		if response, err := instanceAPI.ListServers(&instance.ListServersRequest{Zone: zone}); err == nil {
			all.TotalCount += response.TotalCount
			all.Servers = append(all.Servers, response.Servers...)
		} else {
			printErr("ListServers %v: %v", zone, err)
		}
	}
	sw.parseNewServers(all)
}

func (sw *scalewayWorker) parseNewServers(response *instance.ListServersResponse) {
	servers := map[serverID]*serverInfo{}
	serversList := []serverID{}
	sw.servers.L.RLock()
	sw.servers.ServersList = nil
	for _, item := range response.Servers {
		id := serverID(item.ID)
		if _, ok := servers[id]; ok {
			continue
		}
		serversList = append(serversList, id)
		servers[id] = &serverInfo{item.ID, item.Name, "IPv4", "IPv6", item.State.String(),
			"REGION", false, false, false, "PING"}

		if item.PublicIP != nil {
			servers[id].IPv4 = item.PublicIP.Address.String()
			servers[id].isIPv4 = true
		}
		if item.IPv6 != nil {
			servers[id].IPv6 = item.IPv6.Address.String()
			servers[id].isIPv6 = true
		}
		if item.Location != nil {
			servers[id].REGION = item.Location.ZoneID
		}
		if old, ok := sw.servers.D[id]; ok {
			servers[id].pingState = old.pingState
			servers[id].pingMS = old.pingMS
		}
	}
	oldSize := len(sw.servers.ServersList)
	newSize := len(serversList)
	sw.servers.L.RUnlock()
	sw.servers.L.Lock()
	sw.servers.D = servers
	sw.servers.ServersList = serversList
	sw.servers.L.Unlock()
	sw.updateMenu(sw.getViewMask(), oldSize != newSize)
}

func (sw *scalewayWorker) getViewMask() string {
	sw.config.L.RLock()
	mask := sw.config.D.ViewMask
	sw.config.L.RUnlock()
	return mask
}

func (sw *scalewayWorker) WriteToClipboard(idx int) (err error) {
	sw.config.L.RLock()
	mask := sw.config.D.CopyMask
	sw.config.L.RUnlock()

	sw.servers.L.RLock()
	defer sw.servers.L.RUnlock()
	count := len(sw.servers.ServersList)
	if 0 <= idx && idx < count && count != 0 {
		id := sw.servers.ServersList[idx]
		if item, ok := sw.servers.D[id]; ok {
			err = clipboard.WriteAll(fillMask(mask, item))
		} else {
			panic(fmt.Errorf(""))
		}
	}
	return
}

func (sw *scalewayWorker) Quit() {
	select {
	case sw.stopChan <- syscall.SIGTERM:
	default:
	}
}

func (sw *scalewayWorker) CFGChange(id cfgActionID) {
	select {
	case sw.signalsChan <- id:
	default:
	}
}
