package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/OpenPeeDeeP/xdg"
)

const cfgName = "settings.json"

type settingsData struct {
	OrganizationID string `json:"organization_id"`
	AccessKey      string `json:"access_key"`
	SecretKey      string `json:"secret_key"`

	ViewMask string `json:"view_mask"`
	CopyMask string `json:"copy_mask"`

	CheckInterval int `json:"check_interval"`
	PingInterval  int `json:"ping_interval"`
}

type settingsStorage struct {
	D *settingsData
	L sync.RWMutex
}

func (s *settingsStorage) ResetSettings() {
	s.L.Lock()
	defer s.L.Unlock()
	*s.D = *makeDefaultSettingsData()
}

func (s *settingsStorage) LoadFrom(path string) (err error) {
	if result, err := loadFromFS(path); err == nil {
		s.L.Lock()
		defer s.L.Unlock()
		*s.D = *result
	}
	return
}

func (s *settingsStorage) SaveTo(path string) error {
	s.L.RLock()
	defer s.L.RUnlock()
	return saveToFS(path, s.D)
}

// Save to settings.json
func (s *settingsStorage) Save() error {
	cfgHome := xdg.New("", appName).ConfigHome()
	err := os.Mkdir(cfgHome, 0700)
	if err == nil || os.IsExist(err) {
		cfgPath := filepath.Join(cfgHome, cfgName)
		return s.SaveTo(cfgPath)
	}
	return err
}

// load from settings.json or make default
func makeSettingsStorage() *settingsStorage {
	data := settingsStorage{D: makeDefaultSettingsData()}
	if cfgPath := xdg.New("", appName).QueryConfig(cfgName); cfgPath != "" {
		_ = data.LoadFrom(cfgPath)
	}
	return &data
}

func makeDefaultSettingsData() *settingsData {
	result := settingsData{}
	result.ViewMask = "{ALIVE} {FLAG} {NAME} {IPvX} {STATE}"
	result.CopyMask = "ssh root@{IPv4}"
	result.CheckInterval = 1200
	result.PingInterval = 10
	return &result
}

func loadFromFS(path string) (*settingsData, error) {
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("Error get file state %s: %v", path, err)
		}
		return nil, fmt.Errorf("File not found: %s", path)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s: %v", path, err)
	}
	result := settingsData{}
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("JSON Unmarshal error %s: %v", path, err)
	}
	return &result, nil
}

func saveToFS(path string, data *settingsData) error {
	result, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("JSON Marshal error: %v", err)
	}
	if err = ioutil.WriteFile(path, result, 0600); err != nil {
		err = fmt.Errorf("Saving error %s: %v", path, err)
	}
	return err
}
