package provider

import (
	"log/slog"
	"sort"
)

var providersList []Provider
var providersLookup map[ModelProviderName]Provider

// DefaultProviders returns the built-in provider configurations.
func DefaultProviders() []Provider {
	return []Provider{
		{
			Name:         ModelProviderAnthropic,
			Model:        DefaultAnthropicModel,
			AuthEnvVars:  []string{"CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY"},
			AllowDomains: []string{".anthropic.com"},
		},
		{
			Name:         ModelProviderOpenAI,
			Model:        DefaultOpenAIModel,
			AuthEnvVars:  []string{"CODEX_API_KEY", "OPENAI_API_KEY"},
			AllowDomains: []string{".openai.com"},
		},
	}
}

// RegisterDefaultProviders registers the built-in provider configurations.
func RegisterDefaultProviders() {
	RegisterProviders(DefaultProviders())
}

// RegisterProviders wires up the given provider configurations.
// No automatic initialization happens, which allows keeping the module easily configurable
// and testable without mandatory default values.
func RegisterProviders(providers []Provider) {
	providersList = providers
	providersLookup = make(map[ModelProviderName]Provider, len(providers))
	for _, p := range providers {
		providersLookup[p.Name] = p
	}
}

// Names ProviderNames returns the registered provider names in sorted order.
func Names() []string {
	var names []string
	for _, p := range providersList {
		names = append(names, p.Name.String())
	}
	sort.Strings(names)
	return names
}

// Lookup returns the provider registered under name.
func Lookup(provider ModelProviderName) Provider {
	p, ok := providersLookup[provider]
	if !ok {
		slog.Warn("requested provider is not found; using default value", "value", DefaultModelProvider.String())
		p = providersLookup[DefaultModelProvider]
	}
	return p
}
