package config

import "fmt"

// ContinuePolicy controls how a task resumes a previously started workspace.
type ContinuePolicy int

const (
	// ContinuePolicyDefault leaves the decision to the caller; it behaves like ContinuePolicyRestart.
	ContinuePolicyDefault ContinuePolicy = iota
	// ContinuePolicyRestart discards any existing workspace and starts the task fresh.
	ContinuePolicyRestart
	// ContinuePolicyResume reuses an existing workspace instead of starting over.
	ContinuePolicyResume
)

// continuePolicyNames maps the YAML string representation to its ContinuePolicy value.
var continuePolicyNames = map[string]ContinuePolicy{
	"":        ContinuePolicyDefault,
	"default": ContinuePolicyDefault,
	"restart": ContinuePolicyRestart,
	"resume":  ContinuePolicyResume,
}

// MarshalYAML encodes the policy as its lowercase string name, defaulting to "default".
func (p ContinuePolicy) MarshalYAML() (interface{}, error) {
	for name, val := range continuePolicyNames {
		if val == p && name != "" {
			return name, nil
		}
	}
	return "default", nil
}

// UnmarshalYAML decodes the policy from its string name, returning an error for unknown values.
func (p *ContinuePolicy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	v, ok := continuePolicyNames[s]
	if !ok {
		return fmt.Errorf("invalid continue_policy: %q", s)
	}
	*p = v
	return nil
}
