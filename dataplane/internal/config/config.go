// Package config — конфиги сервера и клиента в формате TOML.
package config

import (
	"fmt"
	"net/netip"
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/<login>/sibtunnel/dataplane/internal/keys"
)

// Server — /etc/sibtunnel/server.toml
type Server struct {
	Interface struct {
		PrivateKeyFile string `toml:"private_key_file"`
		ListenPort     int    `toml:"listen_port"`
		TunName        string `toml:"tun_name"`
		Address        string `toml:"address"` // "10.77.0.1/22"
		MTU            int    `toml:"mtu"`
		Workers        int    `toml:"workers"` // фаза 4: число конвейеров
	} `toml:"interface"`
	UAPI struct {
		Socket string `toml:"socket"`
	} `toml:"uapi"`
	Metrics struct {
		Listen string `toml:"listen"` // "127.0.0.1:9101"
	} `toml:"metrics"`
	Limits struct {
		HandshakesPerIPPerSec float64 `toml:"handshakes_per_ip_per_sec"`
	} `toml:"limits"`

	PrivateKey keys.Key     `toml:"-"`
	Prefix     netip.Prefix `toml:"-"`
}

// Client — файл, который пользователь скачивает по приглашению.
type Client struct {
	Interface struct {
		PrivateKey string   `toml:"private_key"`
		Address    string   `toml:"address"`
		DNS        []string `toml:"dns"`
		MTU        int      `toml:"mtu"`
	} `toml:"interface"`
	Server struct {
		PublicKey        string `toml:"public_key"`
		Endpoint         string `toml:"endpoint"`
		KeepaliveSeconds int    `toml:"keepalive_seconds"`
	} `toml:"server"`
}

func LoadServer(path string) (*Server, error) {
	var c Server
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	if c.Interface.ListenPort == 0 {
		c.Interface.ListenPort = 51900
	}
	if c.Interface.MTU == 0 {
		c.Interface.MTU = 1420
	}
	if c.Interface.TunName == "" {
		c.Interface.TunName = "sib0"
	}
	if c.Interface.Workers == 0 {
		c.Interface.Workers = 1
	}
	if c.Limits.HandshakesPerIPPerSec == 0 {
		c.Limits.HandshakesPerIPPerSec = 5
	}
	raw, err := os.ReadFile(c.Interface.PrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("config: private key: %w", err)
	}
	if c.PrivateKey, err = keys.Parse(strings.TrimSpace(string(raw))); err != nil {
		return nil, err
	}
	if c.Prefix, err = netip.ParsePrefix(c.Interface.Address); err != nil {
		return nil, fmt.Errorf("config: address: %w", err)
	}
	return &c, nil
}

func LoadClient(path string) (*Client, error) {
	var c Client
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	if c.Interface.MTU == 0 {
		c.Interface.MTU = 1420
	}
	if c.Server.KeepaliveSeconds == 0 {
		c.Server.KeepaliveSeconds = 25
	}
	return &c, nil
}
