// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
)

func TestTemplate_SquidConf(t *testing.T) {
	t.Parallel()
	data := Data{
		ProxyPort: 3128,
	}
	out, err := SquidConf(&data)
	assert.NotErr(t, err)
	s := string(out)
	assert.ContainsAll(t, s, "http_port 3128", "acl allowlist dstdomain", "http_access deny all")
}

func TestTemplate_SettingsJSON(t *testing.T) {
	t.Parallel()
	data := Data{
		Permissions: []string{
			"Bash(git:*)",
			"Read(./**)",
			"Write(./src/**)",
		},
	}
	out, err := SettingsJSON(&data)
	assert.NotErr(t, err)
	s := string(out)
	assert.ContainsAll(t, s, data.Permissions...)
	assert.Contains(t, s, `"permissions"`)
	assert.Contains(t, s, `"allow"`)
}

func TestTemplate_GuidelinesMD(t *testing.T) {
	t.Parallel()
	data := Data{
		AllowedDomains: []string{
			"registry.npmjs.org",
			"github.com",
			"developer.mozilla.org",
		},
	}
	out, err := GuidelinesMD(&data)
	assert.NotErr(t, err)
	s := string(out)
	assert.ContainsAll(t, s, data.AllowedDomains...)
	assert.Contains(t, s, "Autonomous Operation Mode")
	assert.Contains(t, s, "ALLOWLISTED DOMAINS")
}

func TestTemplate_Allowlist(t *testing.T) {
	t.Parallel()
	data := Data{
		AllowedDomains: []string{
			".npmjs.org",
			".github.com",
			"developer.mozilla.org",
		},
	}
	out, err := Allowlist(&data)
	assert.NotErr(t, err)
	s := string(out)
	assert.ContainsAll(t, s, data.AllowedDomains...)
}

func TestTemplate_WriteAll(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	t.Logf("temp dir: %s", tmpDir)
	data := Data{
		WorkspaceID:    "test-all",
		WorkspaceName:  "ws-1",
		Model:          "best-model",
		MaxTurns:       999,
		ProxyIP:        "1.1.1.1",
		ProxyPort:      8888,
		AllowedDomains: []string{"example.com", "python.org"},
		GitUserName:    "test user",
		GitUserEmail:   "git@test-user.com",
		Permissions:    nil,
		Env: map[string]string{
			"one": "1",
			"two": "2",
		},
	}
	assert.NotErr(t, WriteAll(tmpDir, &data))
	for _, name := range []string{
		filepath.Join("proxy", "squid.conf"),
		filepath.Join("proxy", "allowlist.txt"),
		"settings.json",
		"CLAUDE.md",
	} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err != nil {
			t.Errorf("expected %s to be generated: %v", name, err)
		}
	}
}
