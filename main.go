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

func isPublicAsset(path string) bool {
	switch path {
	case "/login.html", "/style.css", "/favicon.png", "/app-core.js", "/theme.js":
		return true
	default:
		return false
	}
}

func ensureSessionSecret(cfg *config.Config, path string) error {
	if cfg.SessionSecret != "" {
		if len([]byte(cfg.SessionSecret)) < 32 {
			return fmt.Errorf("SESSION_SECRET must be at least 32 bytes")
		}
		return config.SecureConfigFile(path)
	}
	secret, err := auth.GenerateSessionSecret()
	if err != nil {
		return err
	}
	if err := config.SaveConfigValue(path, "SESSION_SECRET", secret); err != nil {
		return err
	}
	cfg.SessionSecret = secret
	return nil
}

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
			hash, err := auth.HashPassword(strings.TrimSpace(pass))
			if err != nil {
				log.Fatalf("Failed to hash password: %v", err)
			}
			if err := config.SaveConfigValue(*configPathPtr, "AUTH_PASS_HASH", hash); err != nil {
				log.Fatalf("Failed to update password: %v", err)
			}
			fmt.Println("Password updated successfully.")
		}
		if *setDirPtr != "" {
			if err := config.SaveConfigValue(*configPathPtr, "WORKSPACE_DIR", *setDirPtr); err != nil {
				log.Fatalf("Failed to update workspace directory: %v", err)
			}
			fmt.Println("Workspace directory updated.")
		}
		if *setUserPtr != "" {
			if err := config.SaveConfigValue(*configPathPtr, "AUTH_USER", *setUserPtr); err != nil {
				log.Fatalf("Failed to update username: %v", err)
			}
			fmt.Println("Username updated.")
		}
		if *setBindPtr != "" {
			if err := config.SaveConfigValue(*configPathPtr, "BIND_ADDR", *setBindPtr); err != nil {
				log.Fatalf("Failed to update bind address: %v", err)
			}
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
	if err := ensureSessionSecret(cfg, *configPathPtr); err != nil {
		log.Fatalf("Failed to initialize session secret: %v", err)
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
	mux.HandleFunc("/api/list", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, cfg.SessionSecret, handler.HandleList))
	mux.HandleFunc("/api/months", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, cfg.SessionSecret, handler.HandleMonths))
	mux.HandleFunc("/api/entry", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, cfg.SessionSecret, handler.HandleEntry))
	mux.HandleFunc("/api/search", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, cfg.SessionSecret, handler.HandleSearch))

	// API UNPROTECTED
	mux.HandleFunc("/api/login", handler.HandleLogin)
	mux.HandleFunc("/api/logout", handler.HandleLogout)

	// RAW PROTECTED
	if cfg.WorkspaceDir != "" {
		workspaceRoot, err := os.OpenRoot(cfg.WorkspaceDir)
		if err != nil {
			log.Fatalf("Failed to open workspace: %v", err)
		}
		defer workspaceRoot.Close()
		fsHandler := http.StripPrefix("/raw/", http.FileServerFS(workspaceRoot.FS()))
		mux.HandleFunc("/raw/", auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, cfg.SessionSecret, fsHandler.ServeHTTP))
	}

	// STATIC
	subFS, _ := fs.Sub(webFS, "web")
	staticHandler := http.FileServer(http.FS(subFS))

	// PROTECTED STATIC (except for login.html)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if isPublicAsset(r.URL.Path) {
			staticHandler.ServeHTTP(w, r)
			return
		}
		auth.Middleware(cfg.AuthUser, cfg.AuthPassHash, cfg.SessionSecret, staticHandler.ServeHTTP)(w, r)
	})

	addr := cfg.BindAddr + ":" + strconv.Itoa(cfg.Port)
	fmt.Printf("DailyFlow starting on http://%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
