package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dailyflow_config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.conf")
	content := `
WORKSPACE_DIR = /test/dir
AUTH_USER = admin
# This is a comment
AUTH_PASS_HASH = abc123def
PORT = 8080
BIND_ADDR = 0.0.0.0
`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.WorkspaceDir != "/test/dir" {
		t.Errorf("Expected workspace /test/dir, got %s", cfg.WorkspaceDir)
	}
	if cfg.AuthUser != "admin" {
		t.Errorf("Expected user admin, got %s", cfg.AuthUser)
	}
	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Port)
	}
	if cfg.BindAddr != "0.0.0.0" {
		t.Errorf("Expected bind 0.0.0.0, got %s", cfg.BindAddr)
	}

	// Test missing file
	_, err = LoadConfig("non_existent.conf")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestSaveConfigValue(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dailyflow_save_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.conf")

	// Test creating new
	err = SaveConfigValue(configPath, "KEY1", "VAL1")
	if err != nil {
		t.Fatalf("SaveConfigValue failed: %v", err)
	}

	cfg, _ := LoadConfig(configPath)
	if cfg == nil || cfg.WorkspaceDir != "" { // WorkspaceDir was not set, but KEY1 isn't in struct.
		// Wait, Key1 isn't in Config struct. But LoadConfig only loads known keys.
		// I should use a struct key for testing.
	}

	err = SaveConfigValue(configPath, "WORKSPACE_DIR", "new_dir")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ = LoadConfig(configPath)
	if cfg.WorkspaceDir != "new_dir" {
		t.Errorf("Expected new_dir, got %s", cfg.WorkspaceDir)
	}

	// Test updating
	err = SaveConfigValue(configPath, "WORKSPACE_DIR", "updated_dir")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ = LoadConfig(configPath)
	if cfg.WorkspaceDir != "updated_dir" {
		t.Errorf("Expected updated_dir, got %s", cfg.WorkspaceDir)
	}
}
