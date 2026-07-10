package main

import (
	"dailyflow/internal/api"
	"dailyflow/internal/auth"
	"dailyflow/internal/config"
	"dailyflow/internal/scanner"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

//go:embed web
var webFS embed.FS

func readPasswordInteractively() (string, error) {
	for {
		fmt.Print("Enter new password (will not show): ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Println()
		password := strings.TrimSpace(string(bytePassword))

		fmt.Print("Confirm new password (will not show): ")
		byteConfirm, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Println()
		confirm := strings.TrimSpace(string(byteConfirm))

		if password == "" {
			fmt.Println("Password cannot be empty. Please try again.")
			continue
		}

		if password != confirm {
			fmt.Println("Passwords do not match. Please try again.")
			continue
		}

		return password, nil
	}
}

func main() {
	configPathPtr := flag.String("c", "./config.conf", "Path to the configuration file")
	setPassPtr := flag.Bool("setpass", false, "Interactively set a new password")
	setDirPtr := flag.String("setdir", "", "Set the workspace directory path in config")
	setUserPtr := flag.String("setuser", "", "Set the authentication username in config")
	setBindPtr := flag.String("setbind", "", "Set the listen address in config")

	flag.Parse()

	if *setPassPtr || *setDirPtr != "" || *setUserPtr != "" || *setBindPtr != "" {
		if *setPassPtr {
			pass, err := readPasswordInteractively()
			if err != nil || strings.TrimSpace(pass) == "" {
				log.Fatalf("Invalid password.")
			}
			hash, _ := auth.HashPassword(strings.TrimSpace(pass))
			config.SaveConfigValue(*configPathPtr, "AUTH_PASS_HASH", hash)
			fmt.Println("Password updated successfully.")
		}
		if *setDirPtr != "" {
			config.SaveConfigValue(*configPathPtr, "WORKSPACE_DIR", *setDirPtr)
			fmt.Println("Workspace directory updated.")
		}
		if *setUserPtr != "" {
			config.SaveConfigValue(*configPathPtr, "AUTH_USER", *setUserPtr)
			fmt.Println("Username updated.")
		}
		if *setBindPtr != "" {
			config.SaveConfigValue(*configPathPtr, "BIND_ADDR", *setBindPtr)
			fmt.Println("Bind address updated.")
		}
		return
	}

	cfg, err := config.LoadConfig(*configPathPtr)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to load config: %v", err)
	} else if os.IsNotExist(err) {
		log.Fatalf("Config file not found: %s. Please initialize with -setdir, -setuser, -setpass.", *configPathPtr)
	}

	// Defaults
	if cfg.Port == 0 {
		cfg.Port = 5330
	}
	if cfg.BindAddr == "" {
		cfg.BindAddr = "localhost"
	}
	if cfg.AuthUser == "" {
		cfg.AuthUser = "DailyFlowUser"
	}
	if cfg.AuthPassHash == "" {
		hash, _ := auth.HashPassword("DailyFlowUnsafePasswd")
		cfg.AuthPassHash = hash
	}

	scn := scanner.NewScanner(cfg.WorkspaceDir)
	handler := &api.API{Config: cfg, Scanner: scn}

	// Security Warning
	usingDefaultUser := cfg.AuthUser == "DailyFlowUser"
	usingDefaultPass := auth.CheckPasswordHash("DailyFlowUnsafePasswd", cfg.AuthPassHash)
	if usingDefaultUser || usingDefaultPass {
		fmt.Println("===============================================================")
		fmt.Println(" 💡 SECURITY WARNING: You are using default credentials!")
		if usingDefaultUser {
			fmt.Println(" -> Username is 'DailyFlowUser'. Change it using: ./dailyflow -setuser <name>")
		}
		if usingDefaultPass {
			fmt.Println(" -> Password is 'DailyFlowUnsafePasswd'. Change it using: ./dailyflow -setpass")
		}
		fmt.Println("===============================================================")
	}

	mux := http.NewServeMux()

	// API PROTECTED
	mux.HandleFunc("/api/list", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, handler.HandleList))
	mux.HandleFunc("/api/search", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, handler.HandleSearch))

	// API UNPROTECTED
	mux.HandleFunc("/api/login", handler.HandleLogin)

	// RAW PROTECTED
	if cfg.WorkspaceDir != "" {
		fsHandler := http.StripPrefix("/raw/", http.FileServer(http.Dir(cfg.WorkspaceDir)))
		mux.HandleFunc("/raw/", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, fsHandler.ServeHTTP))
	}

	// STATIC
	subFS, _ := fs.Sub(webFS, "web")
	staticHandler := http.FileServer(http.FS(subFS))

	// PROTECTED STATIC (except for login.html)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login.html" || r.URL.Path == "/style.css" || r.URL.Path == "/app.js" {
			staticHandler.ServeHTTP(w, r)
			return
		}
		auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, staticHandler.ServeHTTP)(w, r)
	})

	addr := cfg.BindAddr + ":" + strconv.Itoa(cfg.Port)
	fmt.Printf("DailyFlow starting on http://%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
