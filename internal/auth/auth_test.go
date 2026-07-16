package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHashAndPassword(t *testing.T) {
	pass := "mysecretpassword"
	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !CheckPasswordHash(pass, hash) {
		t.Errorf("Password verification failed for correct password")
	}

	if CheckPasswordHash("wrongpassword", hash) {
		t.Errorf("Password verification succeeded for wrong password")
	}
}

func TestGenerateSessionSecret(t *testing.T) {
	first, err := GenerateSessionSecret()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateSessionSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 {
		t.Fatalf("Expected 64 hex characters, got %d", len(first))
	}
	if first == second {
		t.Fatal("Expected independently generated secrets")
	}
}

func TestTokenValidation(t *testing.T) {
	user := "testuser"
	hash := "fakehash"
	secret := "test-session-secret"
	token := GenerateToken(user, hash, secret)

	if !ValidateToken(token, user, hash, secret) {
		t.Errorf("Token validation failed for valid token")
	}

	if ValidateToken(token, "otheruser", hash, secret) {
		t.Errorf("Token validation succeeded for wrong user")
	}

	if ValidateToken("invalid:token:format", user, hash, secret) {
		t.Errorf("Token validation succeeded for invalid format")
	}

	if ValidateToken(token, user, "wronghash", secret) {
		t.Errorf("Token validation succeeded for wrong hash")
	}

	if ValidateToken(token, user, hash, "other-session-secret") {
		t.Errorf("Token validation succeeded for wrong session secret")
	}
}

func TestTokenSupportsUsernamesWithSeparators(t *testing.T) {
	user := "name:with:colons"
	token := GenerateToken(user, "hash", "session-secret")
	if !ValidateToken(token, user, "hash", "session-secret") {
		t.Fatal("Token should support usernames containing colons")
	}
}

func TestMiddleware(t *testing.T) {
	user := "authuser"
	hash := "somehash"
	secret := "middleware-session-secret"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(user, hash, secret, handler)

	// Test case: No cookie
	req := httptest.NewRequest("GET", "/api/list", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing cookie on API, got %d", rr.Code)
	}

	// Test case: Valid cookie
	token := GenerateToken(user, hash, secret)
	req = httptest.NewRequest("GET", "/api/list", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid session, got %d", rr.Code)
	}

	// Test case: UI redirect preserves the same-origin permalink.
	req = httptest.NewRequest("GET", "/?entry=%2F2026%2F07%2Fentry.md", nil)
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected 303 redirect for UI, got %d", rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/login.html?return=%2F%3Fentry%3D%252F2026%252F07%252Fentry.md" {
		t.Errorf("Unexpected login return URL: %s", location)
	}
}
