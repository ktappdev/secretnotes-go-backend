package config

import "testing"

func TestDefaultConfigUsesPBServer(t *testing.T) {
	cfg := Default()

	if len(cfg.Servers) != 1 {
		t.Fatalf("expected 1 default server, got %d", len(cfg.Servers))
	}

	if cfg.Servers[0].URL != "https://pb.secretnotez.com" {
		t.Fatalf("expected default URL to be https://pb.secretnotez.com, got %q", cfg.Servers[0].URL)
	}
	if cfg.DefaultServer != "remote" {
		t.Fatalf("expected default server name to be 'remote', got %q", cfg.DefaultServer)
	}
}
