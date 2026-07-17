package api

import (
	"bytes"
	"dailyflow/internal/auth"
	"dailyflow/internal/config"
	"dailyflow/internal/scanner"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleLogin(t *testing.T) {
	cfg := &config.Config{
		AuthUser:      "user",
		AuthPassHash:  "",
		SessionSecret: "api-test-session-secret",
		CookieSecure:  true,
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
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected one session cookie, got %d", len(cookies))
	}
	if !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("Session cookie is missing security attributes: %#v", cookies[0])
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

func TestHandleLogout(t *testing.T) {
	api := &API{Config: &config.Config{CookieSecure: true}}

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	rr := httptest.NewRecorder()
	api.HandleLogout(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", rr.Code)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != auth.SessionCookieName || cookies[0].MaxAge >= 0 {
		t.Fatalf("Expected logout to expire the session cookie: %#v", cookies)
	}
	if !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("Expired cookie is missing security attributes: %#v", cookies[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/logout", nil)
	rr = httptest.NewRecorder()
	api.HandleLogout(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected 405 for GET logout, got %d", rr.Code)
	}
}

func TestHandleList(t *testing.T) {
	tempDir := t.TempDir()
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
	if rr.Header().Get("X-Has-More") != "false" {
		t.Fatalf("Expected final page header, got %q", rr.Header().Get("X-Has-More"))
	}
}

func TestHandleListReportsMorePages(t *testing.T) {
	tempDir := t.TempDir()
	for i := 0; i < 16; i++ {
		name := filepath.Join(tempDir, fmt.Sprintf("entry-%02d.md", i))
		if err := os.WriteFile(name, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	api := &API{Scanner: scanner.NewScanner(tempDir)}
	req := httptest.NewRequest(http.MethodGet, "/api/list?page=1", nil)
	rr := httptest.NewRecorder()
	api.HandleList(rr, req)

	if rr.Header().Get("X-Has-More") != "true" {
		t.Fatalf("Expected another-page header, got %q", rr.Header().Get("X-Has-More"))
	}
	var results []scanner.JournalEntry
	if err := json.NewDecoder(rr.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 15 {
		t.Fatalf("Expected a full page, got %d", len(results))
	}
}

func TestHandleListFiltersByMonth(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"2026-04-21.md", "2026-03-31.md"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	api := &API{Scanner: scanner.NewScanner(tempDir)}
	req := httptest.NewRequest(http.MethodGet, "/api/list?page=1&month=2026-04", nil)
	rr := httptest.NewRecorder()
	api.HandleList(rr, req)

	var results []scanner.JournalEntry
	if err := json.NewDecoder(rr.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != "/2026-04-21.md" {
		t.Fatalf("Unexpected filtered results: %v", results)
	}
}

func TestHandleMonths(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"2026-04-21.md", "2026-03-31.md"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	api := &API{Scanner: scanner.NewScanner(tempDir)}
	req := httptest.NewRequest(http.MethodGet, "/api/months", nil)
	rr := httptest.NewRecorder()
	api.HandleMonths(rr, req)

	var months []string
	if err := json.NewDecoder(rr.Body).Decode(&months); err != nil {
		t.Fatal(err)
	}
	if len(months) != 2 || months[0] != "2026-04" || months[1] != "2026-03" {
		t.Fatalf("Unexpected months: %v", months)
	}
}

func TestHandleEntry(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "2026-07-17.md"), []byte("# Entry"), 0644); err != nil {
		t.Fatal(err)
	}
	api := &API{Scanner: scanner.NewScanner(tempDir)}

	for _, test := range []struct {
		url  string
		code int
		body string
	}{
		{"/api/entry?path=%2F2026-07-17.md", http.StatusOK, "# Entry"},
		{"/api/entry?path=%2F..%2Foutside.md", http.StatusBadRequest, ""},
		{"/api/entry?path=%2Fmissing.md", http.StatusNotFound, ""},
	} {
		req := httptest.NewRequest(http.MethodGet, test.url, nil)
		rr := httptest.NewRecorder()
		api.HandleEntry(rr, req)
		if rr.Code != test.code {
			t.Errorf("%s: expected %d, got %d", test.url, test.code, rr.Code)
		}
		if test.body != "" && rr.Body.String() != test.body {
			t.Errorf("%s: expected body %q, got %q", test.url, test.body, rr.Body.String())
		}
	}
}

func TestHandleSearch(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("findme   here"), 0644)

	cfg := &config.Config{WorkspaceDir: tempDir}
	scn := scanner.NewScanner(tempDir)
	api := &API{Config: cfg, Scanner: scn}

	req := httptest.NewRequest("GET", "/api/search?q=findme", nil)
	rr := httptest.NewRecorder()
	api.HandleSearch(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	var results []scanner.SearchResult
	json.NewDecoder(rr.Body).Decode(&results)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	req = httptest.NewRequest("GET", "/api/search?q=+++", nil)
	rr = httptest.NewRecorder()
	api.HandleSearch(rr, req)
	results = nil
	json.NewDecoder(rr.Body).Decode(&results)
	if len(results) != 0 {
		t.Errorf("Expected whitespace-only query to return no results, got %d", len(results))
	}
}
