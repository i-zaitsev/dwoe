// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package provider

import (
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"gopkg.in/yaml.v3"
)

func TestProviderNames_EmptyBeforeRegistered(t *testing.T) {
	assert.Zero(t, len(Names()))
}

func TestProvider_Serialization(t *testing.T) {
	tests := []struct {
		inp ModelProviderName
		out string
	}{
		{ModelProviderAnthropic, "anthropic"},
		{ModelProviderOpenAI, "openai"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.inp.String(), tc.out)

		data, err := yaml.Marshal(tc.inp)
		assert.NotErr(t, err)
		assert.Equal(t, strings.TrimSpace(string(data)), tc.out)

		var got ModelProviderName
		assert.NotErr(t, yaml.Unmarshal(data, &got))
		assert.Equal(t, got, tc.inp)
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		name    string
		inp     string
		want    ModelProviderName
		wantErr bool
	}{
		{"anthropic", "anthropic", ModelProviderAnthropic, false},
		{"openai", "openai", ModelProviderOpenAI, false},
		{"empty falls back to default", "", DefaultModelProvider, false},
		{"unknown reports error", "bogus", ModelProviderUnknown, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseName(tc.inp)
			if tc.wantErr {
				assert.Err(t, err)
				return
			}
			assert.NotErr(t, err)
			assert.Equal(t, got, tc.want)
		})
	}
}

func TestRegisterProviders(t *testing.T) {
	t.Cleanup(func() { RegisterProviders(nil) })

	RegisterDefaultProviders()

	assert.NoDiff(t, []string{"anthropic", "openai"}, Names())

	anthropic := Lookup(ModelProviderAnthropic)
	assert.Equal(t, anthropic.Name, ModelProviderAnthropic)
	assert.Equal(t, anthropic.Model, DefaultAnthropicModel)
	assert.NotZero(t, len(anthropic.AuthEnvVars))

	// An unregistered provider falls back to the default one.
	assert.Equal(t, Lookup(ModelProviderUnknown).Name, DefaultModelProvider)
}
