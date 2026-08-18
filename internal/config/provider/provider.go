package provider

import (
	"fmt"
)

// ModelProviderName defines a service which API is used to run agents.
type ModelProviderName int

const (
	ModelProviderUnknown ModelProviderName = iota
	ModelProviderAnthropic
	ModelProviderOpenAI
)

// Balanced everyday models are used as defaults.
// Should be updated as new versions released or the older ones getting deprecated.
const (
	DefaultOpenAIModel    = "gpt-5.6-terra"
	DefaultAnthropicModel = "claude-sonnet-5"
)

func (p ModelProviderName) String() string {
	switch p {
	case ModelProviderAnthropic:
		return "anthropic"
	case ModelProviderOpenAI:
		return "openai"
	default:
		return "unknown"
	}
}

func (p ModelProviderName) MarshalYAML() (interface{}, error) {
	if p == ModelProviderUnknown {
		return nil, nil
	}
	return p.String(), nil
}

// ParseName converts a provider name into its enum value.
// An empty string yields the default provider.
func ParseName(s string) (ModelProviderName, error) {
	switch s {
	case "":
		return DefaultModelProvider, nil
	case "anthropic":
		return ModelProviderAnthropic, nil
	case "openai":
		return ModelProviderOpenAI, nil
	}
	return ModelProviderUnknown, fmt.Errorf("unknown provider name: %s", s)
}

func (p *ModelProviderName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	name, err := ParseName(s)
	if err != nil {
		return err
	}
	*p = name
	return nil
}

// DefaultModelProvider is the agent provider used when none is configured.
const DefaultModelProvider = ModelProviderAnthropic

// Provider describes how to run a given agent backend in a container.
// It holds the data needed to pick an image and model, pass credentials
// through to the container, and allow the provider's API through the proxy.
type Provider struct {
	Name         ModelProviderName `yaml:"provider,omitempty"`
	Model        string            `yaml:"model,omitempty"`
	AuthEnvVars  []string          `yaml:"auth_env_vars,omitempty"`
	AllowDomains []string          `yaml:"allow_domains,omitempty"`
}

func (p *Provider) Unknown() bool {
	return p.Name == ModelProviderUnknown
}

// DefaultModel returns the model a provider uses when none is configured.
// An unknown provider has no default model.
func (p *Provider) DefaultModel() string {
	switch p.Name {
	case ModelProviderAnthropic:
		return DefaultAnthropicModel
	case ModelProviderOpenAI:
		return DefaultOpenAIModel
	default:
		return ""
	}
}
