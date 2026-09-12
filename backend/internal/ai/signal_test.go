package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cryptotrading/internal/models"
)

func TestGenerateSignalIncludesProviderContextOnRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid test key"}}`))
	}))
	defer server.Close()

	client, err := New(Config{
		Provider:          ProviderOpenRouter,
		OpenRouterAPIKey:  "openrouter-test",
		OpenRouterBaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GenerateSignal(context.Background(), SignalRequest{Symbol: "BTCUSDT"})
	if err == nil {
		t.Fatal("expected request error")
	}
	if !strings.Contains(err.Error(), "ai: openrouter generate signal") {
		t.Fatalf("error = %q, want provider context", err)
	}
}

func TestGenerateSignalOpenRouterUsesAnthropicMessagesToolContract(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "ambient-anthropic-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %q, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer openrouter-test" {
			t.Errorf("Authorization = %q, want Bearer auth", got)
		}
		if got := r.Header.Get("X-Api-Key"); got != "" {
			t.Errorf("X-Api-Key = %q, OpenRouter must use Bearer auth", got)
		}

		var request struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.Model != "anthropic/claude-sonnet-5" {
			t.Errorf("request model = %q, want anthropic/claude-sonnet-5", request.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","model":"anthropic/claude-sonnet-5","content":[{"type":"tool_use","id":"toolu_test","name":"emit_trade_signal","input":{"action":"BUY","confidence":0.82,"entry_hint":100,"stop_loss":98,"take_profit":104,"rationale":"test"}}],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client, err := New(Config{
		Provider:          ProviderOpenRouter,
		OpenRouterAPIKey:  "openrouter-test",
		OpenRouterBaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.GenerateSignal(context.Background(), SignalRequest{Symbol: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != models.SignalBuy {
		t.Fatalf("Action = %q, want BUY", result.Action)
	}
	if result.Confidence != 0.82 {
		t.Fatalf("Confidence = %v, want 0.82", result.Confidence)
	}
	if result.EntryHint == nil || *result.EntryHint != 100 {
		t.Fatalf("EntryHint = %v, want 100", result.EntryHint)
	}
	if result.StopLoss == nil || *result.StopLoss != 98 {
		t.Fatalf("StopLoss = %v, want 98", result.StopLoss)
	}
	if result.TakeProfit == nil || *result.TakeProfit != 104 {
		t.Fatalf("TakeProfit = %v, want 104", result.TakeProfit)
	}
	if result.Rationale != "test" {
		t.Fatalf("Rationale = %q, want test", result.Rationale)
	}
	if string(result.RawJSON) != `{"action":"BUY","confidence":0.82,"entry_hint":100,"stop_loss":98,"take_profit":104,"rationale":"test"}` {
		t.Fatalf("RawJSON = %s, want original tool JSON", result.RawJSON)
	}
}
