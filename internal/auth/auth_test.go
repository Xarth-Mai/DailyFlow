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

func TestTokenValidation(t *testing.T) {
	user := "testuser"
	hash := "fakehash"
	token := GenerateToken(user, hash)
	
	if !ValidateToken(token, user, hash) {
		t.Errorf("Token validation failed for valid token")
	}
	
	if ValidateToken(token, "otheruser", hash) {
		t.Errorf("Token validation succeeded for wrong user")
	}
	
	if ValidateToken("invalid:token:format", user, hash) {
		t.Errorf("Token validation succeeded for invalid format")
	}

	if ValidateToken(token, user, "wronghash") {
		t.Errorf("Token validation succeeded for wrong hash")
	}
}

func TestMiddleware(t *testing.T) {
	user := "authuser"
	hash := "somehash"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := Middleware(user, hash, handler)

	// Test case: No cookie
	req := httptest.NewRequest("GET", "/api/list", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing cookie on API, got %d", rr.Code)
	}

	// Test case: Valid cookie
	token := GenerateToken(user, hash)
	req = httptest.NewRequest("GET", "/api/list", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid session, got %d", rr.Code)
	}

	// Test case: UI redirect
	req = httptest.NewRequest("GET", "/", nil)
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected 303 redirect for UI, got %d", rr.Code)
	}
}

func TestTokenExpiration(t *testing.T) {
	// Skeleton for expansion
}
