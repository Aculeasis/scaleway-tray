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

type serversInfo struct {
	D map[serverID]*serverInfo
	L sync.RWMutex
	// for ID position saving
	ServersList []serverID
}

type scalewayWorker struct {
	servers     *serversInfo
	config      *settingsStorage
	menu        *menuPool
	stopChan    chan os.Signal
	signalsChan chan cfgActionID
}

func newScalewayWorker(config *settingsStorage, menu *menuPool) *scalewayWorker {
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
		sw.menu.HideAll()
	}
	sw.servers.L.RLock()
	defer sw.servers.L.RUnlock()
	size := len(sw.servers.ServersList)
	if size > sw.menu.GetSize() {
		size = sw.menu.GetSize()
	}
	for idx := 0; idx < size; idx++ {
		id := sw.servers.ServersList[idx]
		if item, ok := sw.servers.D[id]; ok {
			if ok = sw.menu.UpdateTitle(idx, fillView(mask, item), menuChange); !ok {
				panic(fmt.Errorf("menuPool: Corrupted"))
			}
		} else {
			panic(fmt.Errorf("serversInfo: Corrupted"))
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
	for _, item := range response.Servers {
		id := serverID(item.ID)
		if _, ok := servers[id]; ok {
			continue
		}
		serversList = append(serversList, id)
		servers[id] = &serverInfo{item.ID, item.Name, "IPv4", "IPv6", item.State.String(),
			"REGION", false, false, false, "PING"}
		if old, ok := sw.servers.D[id]; ok {
			servers[id].REGION = old.REGION
			servers[id].pingState = old.pingState
			servers[id].pingMS = old.pingMS
		}

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
	}
	sw.servers.L.RUnlock()

	sw.servers.L.Lock()
	sizeChange := len(sw.servers.ServersList) != len(serversList)
	sw.servers.D = servers
	sw.servers.ServersList = serversList
	sw.servers.L.Unlock()

	sw.updateMenu(sw.getViewMask(), sizeChange)
}

func (sw *scalewayWorker) getViewMask() string {
	sw.config.L.RLock()
	mask := sw.config.D.ViewMask
	sw.config.L.RUnlock()
	return mask
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
