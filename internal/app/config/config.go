package config

import (
	"bufio"
	"strings"

	"github.com/kartFr/Asset-Reuploader/internal/files"
)

var (
	config        = make(map[string]string, 0)
	defaultConfig = map[string]string{
		"port":        "38073",
		"cookie_file": "cookie.txt",
	}
)

func init() {
	contents, err := files.Read("config.ini")
	if err != nil {
		scanner := bufio.NewScanner(strings.NewReader(contents))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			split := strings.Split(line, "=")
			if len(split) != 2 {
				continue
			}

			config[split[0]] = split[1]
		}
	}

	for i, v := range defaultConfig {
		if _, exists := config[i]; exists {
			continue
		}
		config[i] = v
	}
}

func Get(key string) string {
	return config[key]
}
