package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSelectDailyWatchlistOpenRouterUsesAnthropicMessagesToolContract(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","model":"anthropic/claude-sonnet-5","content":[{"type":"tool_use","id":"toolu_test","name":"select_daily_watchlist","input":{"picks":[{"symbol":"BTCUSDT","rationale":"test"}]}}],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":1}}`))
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

	picks, err := client.SelectDailyWatchlist(context.Background(), []WatchlistCandidate{{Symbol: "BTCUSDT"}}, 1, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(picks) != 1 {
		t.Fatalf("len(picks) = %d, want 1", len(picks))
	}
	if picks[0].Symbol != "BTCUSDT" || picks[0].Rationale != "test" {
		t.Fatalf("pick = %+v, want BTCUSDT/test", picks[0])
	}
}
