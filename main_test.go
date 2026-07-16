package main

import (
	"dailyflow/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicAssets(t *testing.T) {
	for path, want := range map[string]bool{
		"/login.html":  true,
		"/style.css":   true,
		"/favicon.png": true,
		"/app.js":      false,
		"/":            false,
		"/app-core.js": true,
		"/theme.js":    true,
	} {
		if got := isPublicAsset(path); got != want {
			t.Errorf("isPublicAsset(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestWorkspaceFileHandlerRejectsEscapingSymlinks(t *testing.T) {
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(filepath.Join(workspace, "inside.txt"), []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "escape.txt")); err != nil {
		t.Fatal(err)
	}

	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	handler := http.FileServerFS(root.FS())

	for path, wantOK := range map[string]bool{"/inside.txt": true, "/escape.txt": false} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if gotOK := rr.Code == http.StatusOK; gotOK != wantOK {
			t.Errorf("GET %s returned %d", path, rr.Code)
		}
	}
}

func TestEnsureSessionSecretRejectsWeakConfiguredValue(t *testing.T) {
	cfg := &config.Config{SessionSecret: "too-short"}
	if err := ensureSessionSecret(cfg, filepath.Join(t.TempDir(), "config.conf")); err == nil {
		t.Fatal("Expected a configured session secret shorter than 32 bytes to be rejected")
	}
}

func TestEnsureSessionSecretSecuresExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.conf")
	secret := strings.Repeat("x", 32)
	if err := os.WriteFile(path, []byte("SESSION_SECRET="+secret+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ensureSessionSecret(&config.Config{SessionSecret: secret}, path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("Expected config permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestEnsureSessionSecretPersistsGeneratedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.conf")
	if err := os.WriteFile(path, []byte("AUTH_USER=test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}

	if err := ensureSessionSecret(cfg, path); err != nil {
		t.Fatal(err)
	}
	if len(cfg.SessionSecret) != 64 {
		t.Fatalf("Expected generated secret, got length %d", len(cfg.SessionSecret))
	}

	reloaded, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.SessionSecret != cfg.SessionSecret {
		t.Fatal("Generated session secret was not persisted")
	}
}
