// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package template renders embedded Go templates for container configuration files.
package template

import (
	"bytes"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	squidConfTmpl    = template.Must(template.ParseFS(templateFS, "templates/squid.conf.tmpl"))
	allowlistTmpl    = template.Must(template.ParseFS(templateFS, "templates/allowlist.txt.tmpl"))
	settingsJSONTmpl = template.Must(template.ParseFS(templateFS, "templates/settings.json.tmpl"))
	guidelinesMDTmpl = template.Must(template.ParseFS(templateFS, "templates/CLAUDE.md.tmpl"))
)

// Data holds the values passed to all workspace templates.
type Data struct {
	WorkspaceID    string
	WorkspaceName  string
	Model          string
	MaxTurns       int
	ProxyIP        string
	ProxyPort      int
	AllowedDomains []string
	GitUserName    string
	GitUserEmail   string
	Env            map[string]string
	Permissions    []string
}

// SquidConf renders the Squid proxy configuration.
func SquidConf(data *Data) ([]byte, error) {
	return render(squidConfTmpl, data)
}

// Allowlist renders the proxy domain allowlist.
func Allowlist(data *Data) ([]byte, error) {
	return render(allowlistTmpl, data)
}

// SettingsJSON renders the Claude Code settings file.
func SettingsJSON(data *Data) ([]byte, error) {
	return render(settingsJSONTmpl, data)
}

// GuidelinesMD renders the CLAUDE.md guidelines file.
func GuidelinesMD(data *Data) ([]byte, error) {
	return render(guidelinesMDTmpl, data)
}

// WriteAll renders and writes all workspace templates to basePath.
func WriteAll(basePath string, data *Data) error {
	slog.Info("template: write-all", "path", basePath)
	templates := []struct {
		name   string
		render func(*Data) ([]byte, error)
		mode   os.FileMode
	}{
		{filepath.Join("proxy", "squid.conf"), SquidConf, 0o644},
		{filepath.Join("proxy", "allowlist.txt"), Allowlist, 0o644},
		{"settings.json", SettingsJSON, 0o644},
		{"CLAUDE.md", GuidelinesMD, 0o644},
	}
	for _, t := range templates {
		out, tErr := t.render(data)
		if tErr != nil {
			return fmt.Errorf("template: rendering %s: %w", t.name, tErr)
		}
		path := filepath.Join(basePath, t.name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("template: mkdir for %s: %w", t.name, err)
		}
		if fErr := os.WriteFile(path, out, t.mode); fErr != nil {
			return fmt.Errorf("template: file writing %s: %w", t.name, fErr)
		}
	}
	return nil
}

func render(tmpl *template.Template, data *Data) ([]byte, error) {
	slog.Debug("template: render", "name", tmpl.Name())
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("template %s: %w", tmpl.Name(), err)
	}
	return buf.Bytes(), nil
}
