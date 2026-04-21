package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	WorkspaceDir string
	AuthUser     string
	AuthPassHash string
	Port         int
	BindAddr     string
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}

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
		val := strings.TrimSpace(parts[1])

		switch key {
		case "WORKSPACE_DIR":
			config.WorkspaceDir = val
		case "AUTH_USER":
			config.AuthUser = val
		case "AUTH_PASS_HASH":
			config.AuthPassHash = val
		case "PORT":
			if parsed, err := strconv.Atoi(val); err == nil {
				config.Port = parsed
			}
		case "BIND_ADDR":
			config.BindAddr = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

func SaveConfigValue(path, key, value string) error {
	file, err := os.Open(path)
	if err != nil {
		// If file doesn't exist, we create it
		if os.IsNotExist(err) {
			content := fmt.Sprintf("%s=%s\n", key, value)
			return os.WriteFile(path, []byte(content), 0600)
		}
		return fmt.Errorf("config file inaccessible: %w", err)
	}

	var lines []string
	found := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
			found = true
		} else {
			lines = append(lines, line)
		}
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0600)
}
