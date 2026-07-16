package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookieName = "dailyflow_session"
)

func GenerateSessionSecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return hex.EncodeToString(secret), nil
}

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

type tokenPayload struct {
	Username string `json:"u"`
	Expires  int64  `json:"e"`
}

// GenerateToken creates a signed token for a username.
func GenerateToken(username, passwordHash, sessionSecret string) string {
	payload, _ := json.Marshal(tokenPayload{
		Username: username,
		Expires:  time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, []byte(sessionSecret+passwordHash))
	mac.Write([]byte(encodedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return encodedPayload + "." + signature
}

// ValidateToken checks if a token is valid and not expired.
func ValidateToken(token, expectedUser, passwordHash, sessionSecret string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(sessionSecret+passwordHash))
	mac.Write([]byte(parts[0]))
	providedSignature, err := hex.DecodeString(parts[1])
	if err != nil || !hmac.Equal(providedSignature, mac.Sum(nil)) {
		return false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var payload tokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false
	}
	return payload.Username == expectedUser && time.Now().Unix() < payload.Expires
}

func NewSessionCookie(token string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60,
	}
}

func ExpiredSessionCookie(secure bool) *http.Cookie {
	cookie := NewSessionCookie("", secure)
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(1, 0)
	return cookie
}

// Middleware verifies the session cookie.
func Middleware(username, passwordHash, sessionSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || !ValidateToken(cookie.Value, username, passwordHash, sessionSecret) {
			// If request is for API, return 401
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/raw/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Otherwise redirect to login and preserve the same-origin target.
			returnPath := url.QueryEscape(r.URL.RequestURI())
			http.Redirect(w, r, "/login.html?return="+returnPath, http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
