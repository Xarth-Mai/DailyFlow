package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookieName = "dailyflow_session"
	// In a real app, this should be in config/env
	sessionSecret = "dailyflow-secret-key-change-me"
)

// HashPassword generates a bcrypt hash for a password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPasswordHash verifies a password against a bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a simple signed token for a username.
func GenerateToken(username, passwordHash string) string {
	expire := time.Now().Add(30 * 24 * time.Hour).Unix()
	payload := fmt.Sprintf("%s:%d", username, expire)
	
	mac := hmac.New(sha256.New, []byte(sessionSecret+passwordHash))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	
	return fmt.Sprintf("%s:%s", payload, signature)
}

// ValidateToken checks if a token is valid and not expired.
func ValidateToken(token, expectedUser, passwordHash string) bool {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return false
	}
	
	username := parts[0]
	expireStr := parts[1]
	signature := parts[2]
	
	if username != expectedUser {
		return false
	}
	
	payload := fmt.Sprintf("%s:%s", username, expireStr)
	mac := hmac.New(sha256.New, []byte(sessionSecret+passwordHash))
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	
	if signature != expectedSignature {
		return false
	}
	
	var expire int64
	fmt.Sscanf(expireStr, "%d", &expire)
	return time.Now().Unix() < expire
}

// Middleware verifies the session cookie.
func Middleware(username, passwordHash string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || !ValidateToken(cookie.Value, username, passwordHash) {
			// If request is for API, return 401
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/raw/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Otherwise redirect to login
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
