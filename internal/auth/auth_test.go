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
	token := GenerateToken(user)
	
	if !ValidateToken(token, user) {
		t.Errorf("Token validation failed for valid token")
	}
	
	if ValidateToken(token, "otheruser") {
		t.Errorf("Token validation succeeded for wrong user")
	}
	
	if ValidateToken("invalid:token:format", user) {
		t.Errorf("Token validation succeeded for invalid format")
	}
}

func TestMiddleware(t *testing.T) {
	user := "authuser"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := Middleware(user, handler)

	// Test case: No cookie
	req := httptest.NewRequest("GET", "/api/list", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing cookie on API, got %d", rr.Code)
	}

	// Test case: Valid cookie
	token := GenerateToken(user)
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
