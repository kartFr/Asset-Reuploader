package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	currentConfig = map[string]string{
		"cookie_file": "cookie.txt",
		"port":        "8976",
		"api_key":     "",       // <-- NEW
	}
	configPath = "config.ini"
)

// Get reads a config value. Returns empty string if not found.
func Get(key string) string {
	loadConfig()
	return currentConfig[key]
}

// Set updates a config value in memory.
func Set(key, value string) {
	currentConfig[key] = strings.TrimSpace(value)
}

// Save writes the in‑memory config back to config.ini.
func Save() error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer file.Close()

	for key, value := range currentConfig {
		_, err := fmt.Fprintf(file, "%s=%s\n", key, value)
		if err != nil {
			return fmt.Errorf("write config: %w", err)
		}
	}
	return nil
}

// loadConfig reads config.ini and merges its values into currentConfig.
func loadConfig() {
	file, err := os.Open(configPath)
	if err != nil {
		return // use defaults
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		currentConfig[key] = value
	}
}
