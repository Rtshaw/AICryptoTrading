# OpenRouter Provider Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add safe Anthropic/OpenRouter provider selection to the existing AI layer while preserving both structured tool-call flows and leaving autotrading disabled.

**Architecture:** Keep one `anthropic.Client` inside `ai.Client`; provider-specific options are selected once in `ai.New`. Configuration resolves provider, credentials, model, and base URL centrally, while signal and watchlist code continues sending the same Anthropic Messages requests and parsing the same tool-use blocks.

**Tech Stack:** Go 1.25.11, Anthropic SDK Go v1.71.0, `httptest.Server`, Gin HTTP API, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-09-12-openrouter-provider-migration-design.md`

## Global Constraints

- Supported providers are exactly `anthropic` and `openrouter`; an unset provider defaults to `anthropic`.
- Provider credentials remain separate: `ANTHROPIC_API_KEY` is never used for OpenRouter and `OPENROUTER_API_KEY` is never used for Anthropic.
- OpenRouter's default base URL is `https://openrouter.ai/api`; the SDK must produce a final `/v1/messages` route.
- OpenRouter authentication is `Authorization: Bearer ...`, not `x-api-key`.
- `GenerateSignal` and `SelectDailyWatchlist` keep their existing prompts, tools, forced tool choice, parser, and result contracts.
- Provider failures, missing tool calls, and malformed tool JSON fail closed; no provider fallback is added.
- No automated test uses real Anthropic, OpenRouter, Binance credentials, or real orders.
- The existing `docker-compose.yml` worktree modification is preserved.
- Runtime `autotrade_enabled` must be proven false before integration testing and remain false at handoff.

---

### Task 1: Add provider configuration and client initialization

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/ai/client.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/ai/client_test.go`

**Interfaces:**
- Consumes: environment variables loaded by `config.Load()`.
- Produces: `ai.Config`, `ai.New(ai.Config) (*ai.Client, error)`, `Client.Provider() string`, `Client.Model() string`, and provider-aware `ErrNotConfigured` behavior.

- [ ] **Step 1: Write failing provider-resolution tests**

Add tests that construct `ai.Config` directly and assert:

```go
func TestNewAnthropicProviderUsesAnthropicDefaults(t *testing.T) {
	client, err := New(Config{Provider: "anthropic", AnthropicAPIKey: "anthropic-test"})
	if err != nil { t.Fatal(err) }
	if client.Provider() != "anthropic" || client.Model() != "claude-sonnet-5" || !client.Enabled() { t.Fatal("unexpected client") }
}

func TestNewOpenRouterProviderUsesOpenRouterDefaults(t *testing.T) {
	client, err := New(Config{Provider: "openrouter", OpenRouterAPIKey: "openrouter-test"})
	if err != nil { t.Fatal(err) }
	if client.Provider() != "openrouter" || client.Model() != "anthropic/claude-sonnet-5" || !client.Enabled() { t.Fatal("unexpected client") }
}

func TestNewMissingProviderKeyDisablesClient(t *testing.T) {
	client, err := New(Config{Provider: "openrouter"})
	if err != nil { t.Fatal(err) }
	if client.Enabled() { t.Fatal("client must be disabled") }
}

func TestNewRejectsInvalidProvider(t *testing.T) {
	_, err := New(Config{Provider: "local"})
	if err == nil { t.Fatal("expected invalid provider error") }
}
```

- [ ] **Step 2: Run the focused tests and verify the expected RED failure**

Run: `cd backend; go test ./internal/ai -run 'TestNew(Anthropic|OpenRouter|Missing|Rejects)' -v`

Expected: compilation/test failure because the provider configuration API and accessors do not exist yet.

- [ ] **Step 3: Implement the minimal provider configuration path**

Add provider constants and the `ai.Config` struct. In `New`, default an empty provider to `anthropic`, resolve provider-specific defaults, reject unknown providers, and construct the SDK with:

```go
option.WithoutEnvironmentDefaults()
option.WithAPIKey(cfg.AnthropicAPIKey)
```

for Anthropic, or:

```go
option.WithoutEnvironmentDefaults()
option.WithBaseURL(openRouterBaseURL)
option.WithAuthToken(cfg.OpenRouterAPIKey)
```

for OpenRouter. Return a disabled client when the selected key is empty. Update `config.Config` and `config.Load()` with the six fields and defaults, then update `main.go` to pass the resolved `ai.Config`, handle constructor errors before starting services, and pass `aiClient.Model()` to model metadata consumers.

- [ ] **Step 4: Run the focused tests and verify GREEN**

Run: `cd backend; go test ./internal/ai -run 'TestNew(Anthropic|OpenRouter|Missing|Rejects)' -v`

Expected: all focused provider tests pass.

- [ ] **Step 5: Refactor only after GREEN**

Centralize provider/model default constants and keep the constructor free of secret values in error strings. Re-run the focused tests.

### Task 2: Add OpenRouter signal transport and parser regression coverage

**Files:**
- Modify: `backend/internal/ai/signal.go`
- Test: `backend/internal/ai/signal_test.go`

**Interfaces:**
- Consumes: `ai.Client` from Task 1 and the existing `SignalRequest`.
- Produces: provider-context errors and an unchanged `SignalResult` from an Anthropic-compatible tool-use response.

- [ ] **Step 1: Write the failing HTTP transport test**

Use `httptest.NewServer` and `New(Config{Provider: "openrouter", OpenRouterAPIKey: "or-test", OpenRouterBaseURL: server.URL})`. Decode the request and assert `r.URL.Path == "/v1/messages"`, `Authorization == "Bearer or-test"`, no `X-Api-Key`, and body model `anthropic/claude-sonnet-5`. Return a message containing:

```json
{"id":"msg_test","type":"message","role":"assistant","model":"anthropic/claude-sonnet-5","content":[{"type":"tool_use","id":"toolu_test","name":"emit_trade_signal","input":{"action":"BUY","confidence":0.82,"entry_hint":100,"stop_loss":98,"take_profit":104,"rationale":"test"}}],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":1}}
```

Call `GenerateSignal` with a minimal `SignalRequest` and assert all six signal values, including the three pointer prices.

- [ ] **Step 2: Run the focused test and verify the expected RED failure**

Run: `cd backend; go test ./internal/ai -run TestGenerateSignalOpenRouter -v`

Expected: failure because the current constructor cannot use OpenRouter and the request is not yet provider-contextualized.

- [ ] **Step 3: Implement the minimal signal changes**

Keep the existing tool schema and parsing intact. Change only request error, parse error, and missing-tool messages to include `c.provider`, preserving `%w` for wrapped errors. Do not change prompts, tool choice, JSON fields, or result validation.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run: `cd backend; go test ./internal/ai -run TestGenerateSignalOpenRouter -v`

Expected: the route, auth, model, no-`x-api-key`, and parsed signal assertions pass.

### Task 3: Add OpenRouter watchlist regression coverage

**Files:**
- Modify: `backend/internal/ai/watchlist_selection.go`
- Test: `backend/internal/ai/watchlist_selection_test.go`

**Interfaces:**
- Consumes: the same provider-selected `ai.Client` used by signal generation.
- Produces: unchanged `[]WatchlistPick` parsing and provider-context errors.

- [ ] **Step 1: Write the failing watchlist transport test**

Use an `httptest.Server` response with a `select_daily_watchlist` tool-use block whose input is `{"picks":[{"symbol":"BTCUSDT","rationale":"test"}]}`. Call `SelectDailyWatchlist` in OpenRouter mode and assert the returned symbol/rationale plus the same route/auth/model/no-`x-api-key` request invariants.

- [ ] **Step 2: Run the focused test and verify the expected RED failure**

Run: `cd backend; go test ./internal/ai -run TestSelectDailyWatchlistOpenRouter -v`

Expected: failure before the shared provider client is wired.

- [ ] **Step 3: Implement the minimal watchlist error-context changes**

Leave the existing tool schema, forced tool choice, and `WatchlistPick` parser unchanged. Add only provider context to upstream, parse, and missing-tool errors.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run: `cd backend; go test ./internal/ai -run TestSelectDailyWatchlistOpenRouter -v`

Expected: all watchlist transport and parser assertions pass.

### Task 4: Update examples and README

**Files:**
- Modify: `.env.example`
- Modify: `README.md`

**Interfaces:**
- Consumes: the final environment variable names and resolved model semantics from Tasks 1-3.
- Produces: copyable direct-Anthropic and OpenRouter setup instructions with no real secrets.

- [ ] **Step 1: Add provider examples**

Document `AI_PROVIDER=anthropic` as the compatibility default, separate Anthropic variables, and separate OpenRouter variables including `OPENROUTER_BASE_URL=`. State that OpenRouter normally uses `https://openrouter.ai/api` and does not need `ANTHROPIC_API_KEY`.

- [ ] **Step 2: Document safety and failure behavior**

Update both language sections to say OpenRouter still uses Claude through the Anthropic Messages API, AI failure prevents automatic order placement, and `autotrade_enabled` must remain false for migration testing. Keep the existing Binance and strategy descriptions unchanged.

- [ ] **Step 3: Read back the docs and verify no secrets**

Run: `rg -n "AI_PROVIDER|ANTHROPIC_API_KEY|OPENROUTER_API_KEY|OPENROUTER_MODEL|OPENROUTER_BASE_URL|autotrade_enabled|OpenRouter" .env.example README.md`

Expected: both provider examples and the safety statements are present; no key value other than empty/example placeholders appears.

### Task 5: Full automated and runtime-safe verification

**Files:**
- Modify only files already listed above if verification exposes a regression.

**Interfaces:**
- Consumes: the complete implementation and documentation.
- Produces: evidence for Go tests, build/vet, Docker startup, provider routing, and disabled autotrading.

- [ ] **Step 1: Run the complete Go test suite**

Run: `cd backend; go test ./...`

Expected: all existing and new tests pass without real network credentials.

- [ ] **Step 2: Run compile/static checks**

Run: `cd backend; go build ./...; go vet ./...`

Expected: both commands exit successfully.

- [ ] **Step 3: Inspect the runtime environment without printing secrets**

Check whether `.env` exists and report only the presence/absence of required key names. If a backend is running, query `http://127.0.0.1:8280/api/settings`; do not proceed to AI integration unless JSON proves `autotrade_enabled:false`. If it is true, use the existing settings endpoint to set it false and read it back.

- [ ] **Step 4: Verify Docker only with autotrading OFF**

Run `docker compose up -d --build`, then inspect `docker compose logs backend` for provider/model metadata and absence of secret values. Query `/api/settings` again and stop the integration run if the actual runtime is not disabled. Do not invoke signal/watchlist endpoints with live credentials unless the local environment is explicitly configured by the user.

- [ ] **Step 5: Record limitations and handoff evidence**

Report exact test results, whether Docker was available, whether manual OpenRouter calls were possible without exposing credentials, and confirm the final runtime state. Do not claim a successful live OpenRouter request or OpenRouter dashboard activity without direct evidence.
