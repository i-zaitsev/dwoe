// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sentinel

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/i-zaitsev/dwoe/internal/config"
	"gopkg.in/yaml.v3"
)

const sentinelFile = ".dwoe-done"

type Sentinel struct {
	hash string
}

func FromConfig(cfg *config.Task) Sentinel {
	hash, _ := taskConfigHash(cfg)
	return Sentinel{hash: hash}
}

func FromDir(baseDir string) Sentinel {
	data, err := os.ReadFile(sentinelPath(baseDir))
	if err != nil {
		return Sentinel{}
	}
	return Sentinel{hash: strings.TrimSpace(string(data))}
}

func (s *Sentinel) Match(cfg *config.Task) bool {
	if s.hash == "" {
		return false
	}
	hash, err := taskConfigHash(cfg)
	if err != nil {
		return false
	}
	return s.hash == hash
}

func (s *Sentinel) Write(baseDir string) error {
	return os.WriteFile(sentinelPath(baseDir), []byte(s.hash), 0o644)
}

func Equal(s1, s2 Sentinel) bool {
	return s1.hash != "" && s1.hash == s2.hash
}

func sentinelPath(basePath string) string {
	return filepath.Join(basePath, sentinelFile)
}

func taskConfigHash(cfg *config.Task) (string, error) {
	cp := *cfg

	// The difference in hash does not come from the differences
	// in task name or continuation policy. It should only be
	// affected by the changes done to other parts of it, like
	// new prompt or agent configuration.
	cp.Name = ""
	cp.ContinuePolicy = config.ContinuePolicyDefault

	data, err := yaml.Marshal(&cp)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
