// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/config"
	"gopkg.in/yaml.v3"
)

// WriteFile writes content to path, creating parent directories as needed.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	MkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// WriteTaskFile serializes a task config to a path as YAML and returns the path.
func WriteTaskFile(t *testing.T, path string, task *config.Task) string {
	t.Helper()
	data, err := yaml.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	WriteFile(t, path, string(data))
	return path
}

// WriteBatchTaskFile writes a task file named after the task into a dir.
func WriteBatchTaskFile(t *testing.T, dir string, task *config.Task) string {
	t.Helper()
	path := filepath.Join(dir, "task-"+task.Name+".yaml")
	return WriteTaskFile(t, path, task)
}

// ReadFile reads the content of a file at the path and returns it as a string.
func ReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
