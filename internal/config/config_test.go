package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "config.conf")
	content := `
WORKSPACE_DIR = /test/dir
AUTH_USER = admin
# This is a comment
AUTH_PASS_HASH = abc123def
SESSION_SECRET = session-secret-value
COOKIE_SECURE = true
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
	if cfg.SessionSecret != "session-secret-value" {
		t.Errorf("Expected session secret to be loaded")
	}
	if !cfg.CookieSecure {
		t.Errorf("Expected secure cookies to be enabled")
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

func TestSecureConfigFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.conf")
	link := filepath.Join(dir, "config.conf")
	if err := os.WriteFile(target, []byte("AUTH_USER=test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := SecureConfigFile(link); err == nil {
		t.Fatal("Expected symlink config to be rejected")
	}
	if err := SaveConfigValue(link, "AUTH_USER", "changed"); err == nil {
		t.Fatal("Expected symlink config update to be rejected")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("Symlink target permissions changed to %o", info.Mode().Perm())
	}
}

func TestSaveConfigValue(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "config.conf")

	err := SaveConfigValue(configPath, "WORKSPACE_DIR", "new_dir")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := LoadConfig(configPath)
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

	if err := os.Chmod(configPath, 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfigValue(configPath, "COOKIE_SECURE", "true"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("Expected config permissions 0600, got %o", info.Mode().Perm())
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.conf" {
		t.Fatalf("Unexpected temporary files after atomic save: %v", entries)
	}
}
