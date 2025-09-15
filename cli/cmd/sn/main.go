package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"secretnotes-cli/internal/api"
	"secretnotes-cli/internal/config"
	"secretnotes-cli/internal/tui"

	"golang.org/x/term"
)

func main() {
	// Flags (overrides)
	var (
		flagURL      string
		flagInsecure bool
		flagAutosave bool
		flagAutosaveMs int
		flagConfigPath string
	)
	flag.StringVar(&flagURL, "url", "", "Server base URL (e.g., http://127.0.0.1:8091)")
	flag.BoolVar(&flagInsecure, "insecure", false, "Skip TLS verification (https only)")
	flag.BoolVar(&flagAutosave, "autosave", false, "Enable autosave")
	flag.IntVar(&flagAutosaveMs, "autosave-debounce-ms", 1200, "Autosave debounce in milliseconds")
	flag.StringVar(&flagConfigPath, "config", "", "Path to config file (optional)")
	flag.Parse()

	// Load or create config
	cfg, cfgPath, err := config.LoadOrInit(flagConfigPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// First-run interactive setup if missing servers
	if len(cfg.Servers) == 0 || cfg.DefaultServer == "" {
		if err := firstRunSetup(&cfg); err != nil {
			log.Fatalf("setup failed: %v", err)
		}
		if err := config.Save(cfgPath, &cfg); err != nil {
			log.Fatalf("failed saving config: %v", err)
		}
	}

	// Apply flag overrides
	if flagURL != "" {
		cfg.OverrideURL(flagURL)
	}
	if flagInsecure {
		cfg.OverrideVerifyTLS(false)
	}
	if flagAutosave {
		cfg.Preferences.AutosaveEnabled = true
	}
	if flagAutosaveMs > 0 {
		cfg.Preferences.AutosaveDebounceMs = flagAutosaveMs
	}

	server := cfg.CurrentServer()
	if server == nil {
		log.Fatal("no server configured")
	}

	// Health check fast-fail
	client := api.NewClient(server.URL, server.VerifyTLS)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.Health(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "warning: server health check failed: %v\n", err)
	}

	// Prompt for passphrase (never saved)
	passphrase, err := promptPassphrase()
	if err != nil {
		log.Fatalf("failed to read passphrase: %v", err)
	}
	// Ensure we zero the buffer on exit
	defer zeroBytes(passphrase)

	// Start TUI editor
	app := tui.NewEditorApp(client,
		passphrase,
		server.Name,
		cfg.Preferences.AutosaveEnabled,
		time.Duration(cfg.Preferences.AutosaveDebounceMs)*time.Millisecond,
	)

	// Handle Ctrl+C as graceful cancel
	ctxRun, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := app.Run(ctxRun); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("app error: %v", err)
	}
}

func promptPassphrase() ([]byte, error) {
	fmt.Print("Passphrase: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}
	if len(b) < 3 {
		return nil, fmt.Errorf("passphrase must be at least 3 characters")
	}
	return b, nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// firstRunSetup asks minimal questions and populates config
func firstRunSetup(cfg *config.Config) error {
	in := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to SecretNotes CLI setup")
	fmt.Println("We'll save non-sensitive preferences to your user config. Your passphrase is never stored.")

	// Server name
	fmt.Print("Server name [local]: ")
	name, _ := in.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "local"
	}

	// URL
	defaultURL := "http://127.0.0.1:8091"
	fmt.Printf("Server URL [%s]: ", defaultURL)
	url, _ := in.ReadString('\n')
	url = strings.TrimSpace(url)
	if url == "" {
		url = defaultURL
	}

	verifyTLS := true
	if strings.HasPrefix(url, "https://") {
		fmt.Print("Verify TLS certificates? [Y/n]: ")
		ans, _ := in.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "n" || ans == "no" {
			verifyTLS = false
		}
	}

	cfg.Servers = []config.Server{{Name: name, URL: url, VerifyTLS: verifyTLS}}
	cfg.DefaultServer = name

	// Preferences
	fmt.Print("Enable autosave? [y/N]: ")
	ans, _ := in.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	cfg.Preferences.AutosaveEnabled = ans == "y" || ans == "yes"

	fmt.Print("Autosave debounce ms [1200]: ")
	debStr, _ := in.ReadString('\n')
	debStr = strings.TrimSpace(debStr)
	if debStr != "" {
		var ms int
		_, err := fmt.Sscanf(debStr, "%d", &ms)
		if err == nil && ms > 100 {
			cfg.Preferences.AutosaveDebounceMs = ms
		}
	}

	return cfg.Validate()
}
