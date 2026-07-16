package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	WorkspaceDir  string
	AuthUser      string
	AuthPassHash  string
	SessionSecret string
	CookieSecure  bool
	Port          int
	BindAddr      string
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
		case "SESSION_SECRET":
			config.SessionSecret = val
		case "COOKIE_SECURE":
			config.CookieSecure, _ = strconv.ParseBool(val)
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
	if key == "" || strings.ContainsAny(key, "=\r\n") || strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("invalid config key or value")
	}

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("config file inaccessible: %w", err)
	}
	if err == nil {
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return statErr
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("config path is not a regular file")
		}
	}

	var lines []string
	if len(content) > 0 {
		lines = strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	}
	found := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[index] = fmt.Sprintf("%s=%s", key, value)
			found = true
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return atomicWriteConfig(path, []byte(strings.Join(lines, "\n")+"\n"))
}

func SecureConfigFile(path string) error {
	file, err := openRegularConfig(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return verifyOpenConfig(path, file)
}

func openRegularConfig(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if err := verifyOpenConfigIdentity(path, file); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func verifyOpenConfig(path string, file *os.File) error {
	if err := verifyOpenConfigIdentity(path, file); err != nil {
		return err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	if fileInfo.Mode().Perm() != 0600 {
		return fmt.Errorf("config permissions are %o, want 600", fileInfo.Mode().Perm())
	}
	return nil
}

func verifyOpenConfigIdentity(path string, file *os.File) error {
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	if !pathInfo.Mode().IsRegular() || !os.SameFile(pathInfo, fileInfo) {
		return fmt.Errorf("config path is not a stable regular file")
	}
	return nil
}

func atomicWriteConfig(path string, content []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	cleanup := func() {
		temp.Close()
		os.Remove(tempPath)
	}

	if err := temp.Chmod(0600); err != nil {
		cleanup()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		cleanup()
		return err
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}
	written, err := openRegularConfig(path)
	if err != nil {
		return err
	}
	if err := verifyOpenConfig(path, written); err != nil {
		written.Close()
		return err
	}
	if err := written.Close(); err != nil {
		return err
	}

	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
