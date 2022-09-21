package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"

	"github.com/brunexgeek/hugye/pkg/dfa"
	"github.com/brunexgeek/hugye/pkg/dns"
)

type ConfigData struct {
	Blacklist  []string `json:"blacklist"`
	Whitelist  []string `json:"whitelist"`
	Monitoring []string `json:"monitoring"`
	Binding    struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	} `json:"binding"`
	ExternalDNS []struct {
		Address string   `json:"address"`
		Name    string   `json:"name"`
		Targets []string `json:"targets"`
	} `json:"external_dns"`
	Cache struct {
		TTL  int `json:"ttl"`
		Size int `json:"size"`
	} `json:"cache"`
}

type Config struct {
	Monitoring  []string
	Binding     *net.UDPAddr
	ExternalDNS []dns.ExternalDNS
	Cache       struct {
		TTL  int
		Size int
	}
	Blocked *dfa.Tree
	Allowed *dfa.Tree
}

func is_ipv4(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 15 {
		return false
	}

	number := 0
	dots := 0
	for _, c := range value {
		if c >= '0' && c <= '9' {
			number++
		} else if c == '.' {
			if number == 0 {
				return false
			}
			number = 0
			dots++
		} else {
			return false
		}
	}
	return dots == 3 && number > 0
}

func LoadRules(file string, tree *dfa.Tree) error {
	fd, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm.Perm())
	if err != nil {
		return err
	}
	reader := bufio.NewReaderSize(fd, 512)

	for true {
		bytes, tl, err := reader.ReadLine()
		if tl {
			return fmt.Errorf("Line too long")
		} else if err != nil {
			return err
		}
		line := strings.Trim(string(bytes), " \t")

		// ignore comments
		if strings.HasPrefix(line, "#") {
			continue
		} else if is_ipv4(line) {
			continue
		} else {
			tree.AddPattern(line)
		}
	}
	return nil
}

func LoadConfig(path string) (*Config, error) {
	config := ConfigData{}

	reader, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm.Perm())
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 1024)
	size, err := reader.Read(buf)
	if size <= 0 {
		return nil, fmt.Errorf("Empty configuration file")
	} else if err != nil {
		return nil, err
	} else {
		buf = buf[:size]
		json.Unmarshal(buf, &config)
	}

	binding, err := netip.ParseAddrPort(fmt.Sprintf("%s:%d", config.Binding.Address, config.Binding.Port))
	if err != nil {
		return nil, err
	}

	result := Config{Binding: net.UDPAddrFromAddrPort(binding)}
	result.ExternalDNS = make([]dns.ExternalDNS, 0, len(config.ExternalDNS))

	for _, item := range config.ExternalDNS {
		addr, err := netip.ParseAddrPort(item.Address + ":53")
		if err != nil {
			continue
		}

		var tree *dfa.Tree
		if len(item.Targets) > 0 {
			tree = dfa.NewTree()
			for _, rule := range item.Targets {
				tree.AddPattern(rule)
			}
		}

		result.ExternalDNS = append(result.ExternalDNS,
			dns.ExternalDNS{Address: net.UDPAddrFromAddrPort(addr), Name: item.Name, Targets: tree})
	}

	result.Blocked = dfa.NewTree()
	if config.Blacklist == nil {
		return nil, fmt.Errorf("Unable to create tree")
	}
	for _, path := range config.Blacklist {
		LoadRules(path, result.Blocked)
	}

	return &result, nil
}
