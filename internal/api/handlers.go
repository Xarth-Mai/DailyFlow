package api

import (
	"dailyflow/internal/auth"
	"dailyflow/internal/config"
	"dailyflow/internal/scanner"
	"encoding/json"
	"net/http"
	"strconv"
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
		token := auth.GenerateToken(data.Username)
		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   2592000, // 30 days
		})
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

func (a *API) HandleList(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	files := a.Scanner.List(page, 15)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (a *API) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results, err := a.Scanner.Search(query)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

