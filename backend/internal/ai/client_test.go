package ai

import "testing"

func TestNewDefaultsToAnthropicProvider(t *testing.T) {
	client, err := New(Config{AnthropicAPIKey: "anthropic-test"})
	if err != nil {
		t.Fatal(err)
	}
	if client.Provider() != "anthropic" {
		t.Fatalf("Provider() = %q, want anthropic", client.Provider())
	}
	if client.Model() != "claude-sonnet-5" {
		t.Fatalf("Model() = %q, want claude-sonnet-5", client.Model())
	}
	if !client.Enabled() {
		t.Fatal("Anthropic client with a key must be enabled")
	}
}

func TestNewAnthropicProviderUsesConfiguredModel(t *testing.T) {
	client, err := New(Config{
		Provider:        "anthropic",
		AnthropicAPIKey: "anthropic-test",
		AnthropicModel:  "claude-test-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.Provider() != "anthropic" {
		t.Fatalf("Provider() = %q, want anthropic", client.Provider())
	}
	if client.Model() != "claude-test-model" {
		t.Fatalf("Model() = %q, want claude-test-model", client.Model())
	}
	if !client.Enabled() {
		t.Fatal("Anthropic client with a key must be enabled")
	}
}

func TestNewOpenRouterProviderUsesOpenRouterDefaults(t *testing.T) {
	client, err := New(Config{Provider: "openrouter", OpenRouterAPIKey: "openrouter-test"})
	if err != nil {
		t.Fatal(err)
	}
	if client.Provider() != "openrouter" {
		t.Fatalf("Provider() = %q, want openrouter", client.Provider())
	}
	if client.Model() != "anthropic/claude-sonnet-5" {
		t.Fatalf("Model() = %q, want anthropic/claude-sonnet-5", client.Model())
	}
	if !client.Enabled() {
		t.Fatal("OpenRouter client with a key must be enabled")
	}
}

func TestNewOpenRouterProviderUsesConfiguredModel(t *testing.T) {
	client, err := New(Config{
		Provider:         "openrouter",
		OpenRouterAPIKey: "openrouter-test",
		OpenRouterModel:  "anthropic/claude-sonnet-4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.Model() != "anthropic/claude-sonnet-4" {
		t.Fatalf("Model() = %q, want configured model", client.Model())
	}
}

func TestNewMissingProviderKeyDisablesClient(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "anthropic", provider: "anthropic"},
		{name: "openrouter", provider: "openrouter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(Config{Provider: tt.provider})
			if err != nil {
				t.Fatal(err)
			}
			if client.Enabled() {
				t.Fatal("client must be disabled without the selected provider key")
			}
		})
	}
}

func TestNewRejectsInvalidProvider(t *testing.T) {
	if _, err := New(Config{Provider: "local", AnthropicAPIKey: "must-not-be-used"}); err == nil {
		t.Fatal("expected invalid provider error")
	}
}

func TestNewRejectsOpenRouterBaseURLThatAlreadyIncludesV1(t *testing.T) {
	_, err := New(Config{
		Provider:          ProviderOpenRouter,
		OpenRouterAPIKey:  "openrouter-test",
		OpenRouterBaseURL: "https://openrouter.ai/api/v1",
	})
	if err == nil {
		t.Fatal("expected base URL validation error")
	}
}

func TestNewRejectsInvalidOpenRouterBaseURL(t *testing.T) {
	_, err := New(Config{
		Provider:          ProviderOpenRouter,
		OpenRouterAPIKey:  "openrouter-test",
		OpenRouterBaseURL: "not-a-url",
	})
	if err == nil {
		t.Fatal("expected base URL validation error")
	}
}
