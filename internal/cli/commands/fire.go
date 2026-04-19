// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/i-zaitsev/dwoe/internal/batch"
	"github.com/i-zaitsev/dwoe/internal/cli"
	"github.com/i-zaitsev/dwoe/internal/config"
)

// cmdFire
type cmdFire struct {
	repo    string // repository URL or local path
	work    string // path to a file or directory with instructions (like task.yaml but any format)
	do      string // inline task prompt (alternative to --work)
	model   string // quick config: model to use
	batchID string // optional batch group ID
}

func newCmdFire() *cmdFire {
	return &cmdFire{}
}

func (c *cmdFire) Name() string { return "fire" }
func (c *cmdFire) Desc() string { return "Quick-start workspace from repo" }
func (c *cmdFire) Args() string { return "--repo <url|path> [flags]" }

func (c *cmdFire) Parse(args []string) error {
	_, err := cli.ParseFlags(c, args, func(fs *flag.FlagSet) {
		fs.StringVar(&c.repo, "repo", "", "repository URL or local path")
		fs.StringVar(&c.repo, "r", "", "repository URL or local path")
		fs.StringVar(&c.work, "work", "", "path to a file or directory with instructions")
		fs.StringVar(&c.work, "w", "", "path to a file or directory with instructions")
		fs.StringVar(&c.do, "do", "", "inline task prompt")
		fs.StringVar(&c.model, "model", "", "model to use")
		fs.StringVar(&c.model, "m", "", "model to use")
		fs.StringVar(&c.batchID, "batch", "", "batch group ID")
		fs.StringVar(&c.batchID, "b", "", "batch group ID")
	})
	if err != nil {
		return err
	}
	if c.repo == "" {
		return cli.CmdErr(c, "%w", &cli.ArgMissingError{Name: "repo"})
	}
	if c.do != "" && c.work != "" {
		return cli.CmdErr(c, "cannot use both --do and --work")
	}
	return nil
}

func (c *cmdFire) Run(e *cli.Env) error {
	slog.Info("cli: fire", "repo", c.repo, "work", c.work, "model", c.model)

	ctx := e.Context()
	global, err := config.LoadGlobalConfig(e.DataDir())
	if err != nil {
		return cli.CmdErr(c, "load global config: %w", err)
	}

	task := &config.Task{
		Agent: config.Agent{
			Model: c.model,
			EnvVars: map[string]string{
				"CLAUDE_CODE_OAUTH_TOKEN": "${CLAUDE_CODE_OAUTH_TOKEN}",
			},
		},
	}

	if c.do != "" {
		task.Agent.TaskPrompt = c.do
	} else if c.work != "" {
		promptFile, err := resolveWork(c.work)
		if err != nil {
			return cli.CmdErr(c, "%w", err)
		}
		task.Source.PromptFile = promptFile
	}
	if isRepoURL(c.repo) {
		task.Source.Repo = c.repo
		task.Source.Branch = "main"
	} else {
		abs, errAbs := filepath.Abs(c.repo)
		if errAbs != nil {
			return cli.CmdErr(c, "resolve repo path: %w", errAbs)
		}
		task.Source.LocalPath = abs
	}

	if task.Git.Name == "" || task.Git.Email == "" {
		name, email := gitIdentity()
		if task.Git.Name == "" {
			task.Git.Name = name
		}
		if task.Git.Email == "" {
			task.Git.Email = email
		}
	}

	if e.NoProxy() {
		task.NoProxy = true
	}
	if e.TaskName() != "" {
		task.Name = e.TaskName()
	}
	config.MergeWithGlobal(task, global)
	task.ApplyDefaults()

	manager, err := e.Manager()
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	ws, err := manager.Create(task)
	if err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	if err := manager.Start(ctx, ws.ID); err != nil {
		return cli.CmdErr(c, "%w", err)
	}

	e.Print("Started workspace: %s\n", ws.Name)
	e.Print("- ID: %s\n", ws.ID)

	if c.batchID != "" {
		rec, errBatch := batch.LoadOrCreate(e.DataDir(), c.batchID, c.repo)
		if errBatch != nil {
			return cli.CmdErr(c, "%w", errBatch)
		}
		rec.Entries = append(rec.Entries, batch.Entry{
			WorkspaceID: ws.ID,
			Branch:      batch.BranchName(c.work),
		})
		if errSave := batch.SaveRecord(e.DataDir(), rec); errSave != nil {
			return cli.CmdErr(c, "%w", errSave)
		}
		e.Print("- Batch: %s\n", c.batchID)
	}

	e.Print("View logs: %s logs %s\n", cli.Prog, ws.ID)

	return nil
}

func gitIdentity() (string, string) {
	name, _ := exec.Command("git", "config", "--global", "user.name").Output()
	email, _ := exec.Command("git", "config", "--global", "user.email").Output()
	return strings.TrimSpace(string(name)), strings.TrimSpace(string(email))
}

func isRepoURL(s string) bool {
	return strings.Contains(s, "://") || strings.HasPrefix(s, "git@")
}

func resolveWork(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("work path: %w", err)
	}
	if !info.IsDir() {
		return filepath.Abs(path)
	}

	content, errCat := catFiles(path)
	if errCat != nil {
		return "", errCat
	}

	tmpName, errTmp := writeTmpFile(content)
	if errTmp != nil {
		return "", errTmp
	}

	return tmpName, nil
}

func catFiles(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("read work dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var buf strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, errRead := os.ReadFile(filepath.Join(path, e.Name()))
		if errRead != nil {
			return "", fmt.Errorf("read %s: %w", e.Name(), errRead)
		}
		_, _ = fmt.Fprintf(&buf, "# %s\n\n%s\n\n---\n\n", e.Name(), string(data))
	}

	return buf.String(), nil
}

func writeTmpFile(content string) (string, error) {
	tmp, errTmp := os.CreateTemp("", "TASK-*.md")
	if errTmp != nil {
		return "", fmt.Errorf("create temp: %w", errTmp)
	}
	defer tmp.Close()
	if _, errW := tmp.WriteString(content); errW != nil {
		return "", fmt.Errorf("write temp: %w", errW)
	}
	return tmp.Name(), nil
}
