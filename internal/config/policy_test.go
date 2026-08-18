// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"gopkg.in/yaml.v3"
)

func TestContinuePolicy_UnmarshalYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		yaml    string
		want    ContinuePolicy
		wantErr bool
	}{
		{`continue_policy: ""`, ContinuePolicyDefault, false},
		{`continue_policy: default`, ContinuePolicyDefault, false},
		{`continue_policy: restart`, ContinuePolicyRestart, false},
		{`continue_policy: resume`, ContinuePolicyResume, false},
		{`continue_policy: bogus`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.yaml, func(t *testing.T) {
			t.Parallel()
			var task Task
			err := yaml.Unmarshal([]byte(tt.yaml), &task)
			if tt.wantErr {
				assert.Err(t, err)
				return
			}
			assert.NotErr(t, err)
			assert.Equal(t, task.ContinuePolicy, tt.want)
		})
	}
}

func TestContinuePolicy_MarshalYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		policy ContinuePolicy
		want   string
	}{
		{ContinuePolicyRestart, "continue_policy: restart"},
		{ContinuePolicyResume, "continue_policy: resume"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			task := Task{ContinuePolicy: tt.policy}
			data, err := yaml.Marshal(&task)
			assert.NotErr(t, err)
			assert.Contains(t, string(data), tt.want)
		})
	}
}

func TestContinuePolicy_MarshalYAML_DefaultOmitted(t *testing.T) {
	t.Parallel()
	task := Task{ContinuePolicy: ContinuePolicyDefault}
	data, err := yaml.Marshal(&task)
	assert.NotErr(t, err)
	assert.ContainsNone(t, string(data), "continue_policy")
}

func TestContinuePolicy_RoundTrip(t *testing.T) {
	t.Parallel()
	for _, policy := range []ContinuePolicy{
		ContinuePolicyDefault,
		ContinuePolicyRestart,
		ContinuePolicyResume,
	} {
		task := Task{ContinuePolicy: policy, Source: Source{LocalPath: "/tmp"}}
		data, err := yaml.Marshal(&task)
		assert.NotErr(t, err)

		var got Task
		assert.NotErr(t, yaml.Unmarshal(data, &got))
		assert.Equal(t, got.ContinuePolicy, policy)
	}
}
