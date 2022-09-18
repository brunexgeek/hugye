package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/brunexgeek/hugye/pkg/dfa"
)

type Config struct {
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
	config := Config{}

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

	config.Blocked = dfa.NewTree()
	if config.Blacklist == nil {
		return nil, fmt.Errorf("Unable to create tree")
	}
	for _, path := range config.Blacklist {
		LoadRules(path, config.Blocked)
	}

	return &config, nil
}
