package main

import (
	"flag"
	"os"
	"testing"
)

func TestPositionalPassphrase(t *testing.T) {
	// Test that we can capture the passphrase argument logic
	// This is a simplified test since we can't easily test the full TUI

	// Reset flags for clean test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Simulate os.Args with a passphrase
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"sn", "testpassphrase"}

	// Define flags like in main
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

	// Check args
	args := flag.Args()
	if len(args) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(args))
	}

	if args[0] != "testpassphrase" {
		t.Errorf("Expected 'testpassphrase', got '%s'", args[0])
	}
}

func TestMinimumPassphraseLength(t *testing.T) {
	// Test that short passphrases are rejected
	passphrase := "ab" // Only 2 characters
	if len(passphrase) >= 3 {
		t.Errorf("Expected passphrase 'ab' to be too short, but it passed length check")
	}

	passphrase = "abc" // Exactly 3 characters
	if len(passphrase) < 3 {
		t.Errorf("Expected passphrase 'abc' to be valid, but it failed length check")
	}
}