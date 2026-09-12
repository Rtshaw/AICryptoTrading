# AICryptoTrading OpenRouter Provider Migration Design

## Goal

Allow the existing AI confirmation layer to use either direct Anthropic or OpenRouter's Anthropic Messages-compatible API without changing trading strategy behavior, structured tool calls, or the autotrader-facing signal contract.

## Current state

- `backend/internal/config/config.go` loads only `ANTHROPIC_API_KEY` and `ANTHROPIC_MODEL`.
- `backend/internal/ai/client.go` owns one `anthropic.Client`, disables itself when the key is empty, and exposes `Enabled()`.
- `signal.go` and `watchlist_selection.go` already share that client and both force an Anthropic tool call, so provider selection can remain entirely inside client initialization.
- `backend/cmd/server/main.go`, `internal/httpapi/ai_handlers.go`, and the signal engine separately receive the Anthropic model string for persistence/logging.
- There are no AI package tests yet.
- The repository has an unrelated existing modification in `docker-compose.yml`; it must remain untouched.

## Design

### Configuration and initialization

`config.Config` will contain separate fields for:

```text
AIProvider
AnthropicAPIKey
AnthropicModel
OpenRouterAPIKey
OpenRouterModel
OpenRouterBaseURL
```

`AI_PROVIDER` resolves to `anthropic` when unset. The provider-specific model defaults are `claude-sonnet-5` and `anthropic/claude-sonnet-5`. The OpenRouter base defaults to `https://openrouter.ai/api`, which lets the Anthropic SDK append `/v1/messages` to produce `https://openrouter.ai/api/v1/messages`.

The AI package receives a configuration struct and returns `(*Client, error)`. Unknown providers fail during startup. A missing key creates a disabled client and an `ErrNotConfigured` error when a call is attempted. The Anthropic SDK is initialized with `option.WithoutEnvironmentDefaults()` so the selected credential cannot be replaced by ambient Anthropic environment/profile configuration. Direct Anthropic uses `option.WithAPIKey`; OpenRouter uses `option.WithAuthToken` and `option.WithBaseURL`.

The client stores provider and resolved model metadata and exposes read-only accessors. No credential or authorization value is logged. Both server model metadata consumers use the resolved client model rather than assuming the Anthropic model.

### Request and error flow

The existing `GenerateSignal` and `SelectDailyWatchlist` request bodies, prompts, tools, forced tool choices, `ToolUseBlock` parsing, and result types remain unchanged. Their upstream and parse errors gain provider context while preserving `%w` wrapping. There is no provider fallback: any request error, missing tool call, or invalid tool JSON remains a hard failure and therefore cannot authorize an automatic order.

```text
config.Load()
    -> ai.New(ai.Config)
        -> one selected Anthropic SDK client
            -> GenerateSignal / SelectDailyWatchlist
                -> existing forced tool-use parser
                    -> existing signal/watchlist consumers
```

### Tests

Unit coverage will verify provider defaults, separate credential selection, disabled behavior, invalid-provider errors, and provider/model metadata. An `httptest.Server` will receive the real SDK request in OpenRouter mode and assert `/v1/messages`, Bearer authorization, the selected model, and absence of `x-api-key`. Mock Anthropic-compatible responses will contain the existing signal and watchlist tool-use blocks, proving both structured parsers still work through OpenRouter transport.

No test calls a real AI provider or Binance. The full Go suite remains the final automated gate.

### Safety and operations

The migration does not alter Binance, strategy, sizing, leverage, stop-loss, take-profit, or position behavior. Before any live/manual integration request, the actual `/api/settings` response must prove `autotrade_enabled=false`; if needed, the runtime setting is explicitly switched off through the existing settings API. The migration never enables it. The integration environment is left disabled. The existing database default and unused `AUTOTRADE_ENABLED_DEFAULT` behavior will be reported rather than silently treated as proof of runtime state.

Documentation will show both provider configurations, explain that OpenRouter still serves Claude through the Anthropic Messages API, and state that AI failures prevent automatic order placement.
