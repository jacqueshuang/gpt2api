package config

import "testing"

func TestLoadInternalMapsOpenAIHostEnv(t *testing.T) {
	t.Setenv("KLEIN_ENV", "dev")
	t.Setenv("KLEIN_SERVER_OPENAI_HOST", "127.0.0.1")

	cfg, err := loadInternal()
	if err != nil {
		t.Fatalf("loadInternal() error = %v", err)
	}
	if cfg.Server.OpenAIHost != "127.0.0.1" {
		t.Fatalf("Server.OpenAIHost = %q, want %q", cfg.Server.OpenAIHost, "127.0.0.1")
	}
}
