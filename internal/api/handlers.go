package api

import (
	"dailyflow/internal/auth"
	"dailyflow/internal/config"
	"dailyflow/internal/scanner"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type API struct {
	Config  *config.Config
	Scanner *scanner.Scanner
}

func (a *API) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if data.Username == a.Config.AuthUser && auth.CheckPasswordHash(data.Password, a.Config.AuthPassHash) {
		token := auth.GenerateToken(data.Username, a.Config.AuthPassHash, a.Config.SessionSecret)
		http.SetCookie(w, auth.NewSessionCookie(token, a.Config.CookieSecure))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

func (a *API) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, auth.ExpiredSessionCookie(a.Config.CookieSecure))
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) HandleList(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	files, hasMore := a.Scanner.ListByMonth(page, 15, r.URL.Query().Get("month"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Has-More", strconv.FormatBool(hasMore))
	json.NewEncoder(w).Encode(files)
}

func (a *API) HandleMonths(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.Scanner.Months())
}

func (a *API) HandleEntry(w http.ResponseWriter, r *http.Request) {
	content, err := a.Scanner.Get(r.URL.Query().Get("path"))
	if errors.Is(err, scanner.ErrInvalidEntryPath) {
		http.Error(w, "Invalid entry path", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(content))
}

func (a *API) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	results := a.Scanner.Search(query)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
