package config

import "testing"

func TestLoadDefaultsToAnthropicProvider(t *testing.T) {
	for _, key := range []string{
		"AI_PROVIDER",
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_MODEL",
		"OPENROUTER_API_KEY",
		"OPENROUTER_MODEL",
		"OPENROUTER_BASE_URL",
	} {
		t.Setenv(key, "")
	}

	cfg := Load()
	if cfg.AIProvider != "anthropic" {
		t.Fatalf("AIProvider = %q, want anthropic", cfg.AIProvider)
	}
	if cfg.AnthropicModel != "claude-sonnet-5" {
		t.Fatalf("AnthropicModel = %q, want claude-sonnet-5", cfg.AnthropicModel)
	}
	if cfg.OpenRouterModel != "anthropic/claude-sonnet-5" {
		t.Fatalf("OpenRouterModel = %q, want anthropic/claude-sonnet-5", cfg.OpenRouterModel)
	}
	if cfg.OpenRouterBaseURL != "" {
		t.Fatalf("OpenRouterBaseURL = %q, want empty optional override", cfg.OpenRouterBaseURL)
	}
}

func TestLoadOpenRouterConfiguration(t *testing.T) {
	t.Setenv("AI_PROVIDER", "openrouter")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-must-stay-separate")
	t.Setenv("ANTHROPIC_MODEL", "anthropic-model")
	t.Setenv("OPENROUTER_API_KEY", "openrouter-test")
	t.Setenv("OPENROUTER_MODEL", "anthropic/claude-sonnet-4")
	t.Setenv("OPENROUTER_BASE_URL", "http://127.0.0.1:9999/api")

	cfg := Load()
	if cfg.AIProvider != "openrouter" {
		t.Fatalf("AIProvider = %q, want openrouter", cfg.AIProvider)
	}
	if cfg.OpenRouterAPIKey != "openrouter-test" {
		t.Fatalf("OpenRouterAPIKey was not loaded")
	}
	if cfg.OpenRouterModel != "anthropic/claude-sonnet-4" {
		t.Fatalf("OpenRouterModel = %q, want configured model", cfg.OpenRouterModel)
	}
	if cfg.OpenRouterBaseURL != "http://127.0.0.1:9999/api" {
		t.Fatalf("OpenRouterBaseURL = %q, want configured URL", cfg.OpenRouterBaseURL)
	}
	if cfg.AnthropicAPIKey != "anthropic-must-stay-separate" || cfg.AnthropicModel != "anthropic-model" {
		t.Fatal("Anthropic configuration must remain separately loaded")
	}
}
