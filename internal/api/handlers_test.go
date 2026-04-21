package api

import (
	"bytes"
	"dailyflow/internal/auth"
	"dailyflow/internal/config"
	"dailyflow/internal/scanner"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleLogin(t *testing.T) {
	cfg := &config.Config{
		AuthUser:     "user",
		AuthPassHash: "",
	}
	hash, _ := auth.HashPassword("pass")
	cfg.AuthPassHash = hash

	api := &API{Config: cfg}

	// Test success
	loginData := map[string]string{"username": "user", "password": "pass"}
	body, _ := json.Marshal(loginData)
	req := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	api.HandleLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	// Test wrong pass
	loginData = map[string]string{"username": "user", "password": "wrong"}
	body, _ = json.Marshal(loginData)
	req = httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	api.HandleLogin(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rr.Code)
	}
}

func TestHandleList(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "api_list_test")
	defer os.RemoveAll(tempDir)
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("content"), 0644)

	cfg := &config.Config{WorkspaceDir: tempDir}
	scn := scanner.NewScanner(tempDir)
	api := &API{Config: cfg, Scanner: scn}

	req := httptest.NewRequest("GET", "/api/list?page=1", nil)
	rr := httptest.NewRecorder()
	api.HandleList(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	var results []scanner.JournalEntry
	json.NewDecoder(rr.Body).Decode(&results)
	if len(results) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(results))
	}
}

func TestHandleSearch(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "api_search_test")
	defer os.RemoveAll(tempDir)
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("findme"), 0644)

	cfg := &config.Config{WorkspaceDir: tempDir}
	scn := scanner.NewScanner(tempDir)
	api := &API{Config: cfg, Scanner: scn}

	req := httptest.NewRequest("GET", "/api/search?q=findme", nil)
	rr := httptest.NewRecorder()
	api.HandleSearch(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	var results []scanner.JournalEntry
	json.NewDecoder(rr.Body).Decode(&results)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}
