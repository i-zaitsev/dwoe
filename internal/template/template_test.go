// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplate_SquidConf(t *testing.T) {
	t.Parallel()
	data := Data{
		ProxyPort: 3128,
	}
	out, err := SquidConf(&data)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"http_port 3128",
		"acl allowlist dstdomain",
		"http_access deny all",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}
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
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range data.Permissions {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if !strings.Contains(s, `"permissions"`) {
		t.Error("output missing permissions key")
	}
	if !strings.Contains(s, `"allow"`) {
		t.Error("output missing allow key")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range data.AllowedDomains {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if !strings.Contains(s, "Autonomous Operation Mode") {
		t.Error("output missing header")
	}
	if !strings.Contains(s, "ALLOWLISTED DOMAINS") {
		t.Error("output missing allowlisted domains section")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range data.AllowedDomains {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}
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
	if err := WriteAll(tmpDir, &data); err != nil {
		t.Fatal(err)
	}
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
