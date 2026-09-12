// Package ai wraps the Anthropic Messages API for trade-signal generation on
// Binance USDS-M perpetual futures. When the selected provider's API key is
// unset, Client.Enabled() is false and callers should surface a "not
// configured" status rather than fail hard - critically, the autotrader
// treats this as a hard stop: it never trades off the bare deterministic rule
// signal alone, so if AI isn't configured, auto-trading is effectively a
// no-op even though it's "live" from launch (see internal/autotrader).
package ai

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	ProviderAnthropic  = "anthropic"
	ProviderOpenRouter = "openrouter"

	defaultAnthropicModel    = "claude-sonnet-5"
	defaultOpenRouterModel   = "anthropic/claude-sonnet-5"
	defaultOpenRouterBaseURL = "https://openrouter.ai/api"
)

var ErrNotConfigured = errors.New("ai: provider credentials are not set")

type Config struct {
	Provider string

	AnthropicAPIKey string
	AnthropicModel  string

	OpenRouterAPIKey  string
	OpenRouterModel   string
	OpenRouterBaseURL string

	SignalSystemPrompt    string
	WatchlistSystemPrompt string
}

type Client struct {
	anthropic anthropic.Client
	provider  string
	model     string
	enabled   bool

	// signalSystemPrompt/watchlistSystemPrompt are the system-level
	// instructions for GenerateSignal/SelectDailyWatchlist respectively -
	// sourced from config (AI_SIGNAL_SYSTEM_PROMPT/AI_WATCHLIST_SYSTEM_PROMPT
	// in .env) so they're editable without a rebuild. See internal/config
	// for the built-in defaults these fall back to when unset.
	signalSystemPrompt    string
	watchlistSystemPrompt string
}

func New(cfg Config) (*Client, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderAnthropic
	}

	client := &Client{
		provider:              provider,
		signalSystemPrompt:    cfg.SignalSystemPrompt,
		watchlistSystemPrompt: cfg.WatchlistSystemPrompt,
	}

	var apiKey string
	var options []option.RequestOption
	switch provider {
	case ProviderAnthropic:
		client.model = cfg.AnthropicModel
		if client.model == "" {
			client.model = defaultAnthropicModel
		}
		apiKey = cfg.AnthropicAPIKey
		options = []option.RequestOption{
			option.WithoutEnvironmentDefaults(),
			option.WithAPIKey(apiKey),
		}
	case ProviderOpenRouter:
		client.model = cfg.OpenRouterModel
		if client.model == "" {
			client.model = defaultOpenRouterModel
		}
		apiKey = cfg.OpenRouterAPIKey
		baseURL := cfg.OpenRouterBaseURL
		baseURL, err := resolveOpenRouterBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		options = []option.RequestOption{
			option.WithoutEnvironmentDefaults(),
			option.WithBaseURL(baseURL),
			option.WithAuthToken(apiKey),
		}
	default:
		return nil, fmt.Errorf("ai: unsupported provider %q", provider)
	}

	if apiKey == "" {
		return client, nil
	}
	client.anthropic = anthropic.NewClient(options...)
	client.enabled = true
	return client, nil
}

func (c *Client) Enabled() bool { return c.enabled }

func (c *Client) Provider() string { return c.provider }

func (c *Client) Model() string { return c.model }

func (c *Client) notConfiguredError() error {
	return fmt.Errorf("ai: %s API key is not set: %w", c.provider, ErrNotConfigured)
}

func resolveOpenRouterBaseURL(raw string) (string, error) {
	if raw == "" {
		raw = defaultOpenRouterBaseURL
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("ai: invalid openrouter base URL: must be an absolute URL without credentials, query, or fragment")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("ai: invalid openrouter base URL: scheme must be http or https")
	}

	path := strings.TrimRight(parsed.Path, "/")
	if path == "/v1" || strings.HasSuffix(path, "/v1") || strings.HasSuffix(path, "/v1/messages") {
		return "", fmt.Errorf("ai: invalid openrouter base URL: omit the /v1/messages path because the SDK appends it")
	}
	parsed.Path = path
	parsed.RawPath = strings.TrimRight(parsed.RawPath, "/")
	return parsed.String(), nil
}
