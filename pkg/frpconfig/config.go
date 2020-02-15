package frpconfig

import (
	"bytes"
	"gopkg.in/ini.v1"
	"sort"
)

// ConfigCommon describes the common section config.
type ConfigCommon struct {
	ServerAddr string `ini:"server_addr"`
	ServerPort int    `ini:"server_port"`
	Token      string `ini:"token,omitempty"`
}

// ConfigApp describes an app config.
type ConfigApp struct {
	Type       string `ini:"type"`
	RemotePort int    `ini:"remote_port"`
	LocalPort  int    `ini:"local_port"`
	LocalAddr  string `ini:"local_ip"`
}

// FrpcConfig describes a frpc configuration.
type FrpcConfig struct {
	Common *ConfigCommon
	Apps   map[string]*ConfigApp
}

// GenerateIni generates frpc ini config.
func (c *FrpcConfig) GenerateIni() (string, error) {
	cfg := ini.Empty()

	secCommon, err := cfg.NewSection("common")
	if err != nil {
		return "", err
	}
	err = secCommon.ReflectFrom(c.Common)
	if err != nil {
		return "", err
	}

	// ensure app sections are sorted
	var appNames []string
	for appName := range c.Apps {
		appNames = append(appNames, appName)
	}
	sort.Strings(appNames)

	for _, appName := range appNames {
		secApp, err := cfg.NewSection(appName)
		if err != nil {
			return "", err
		}
		err = secApp.ReflectFrom(c.Apps[appName])
		if err != nil {
			return "", err
		}
	}

	var b bytes.Buffer
	_, err = cfg.WriteTo(&b)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
