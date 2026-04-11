package sentinel

import (
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/testutil"
)

func TestSentinel_FromConfig(t *testing.T) {
	t.Parallel()
	cfg := config.Task{
		Name:           "test-task",
		Description:    "a test task",
		Source:         config.Source{LocalPath: "/source"},
		ContinuePolicy: config.ContinuePolicyResume,
	}

	sen := FromConfig(&cfg)

	assert.NotEqual(t, sen.hash, "")
}

func TestSentinel_FromFile(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	filePath := baseDir + "/" + sentinelFile
	testHash := "hash-value"
	testutil.WriteFile(t, filePath, testHash)

	sen := FromDir(baseDir)

	assert.Equal(t, sen.hash, testHash)
}

func TestSentinel_SameHash(t *testing.T) {
	t.Parallel()

	// the configs are considered to be the same if the only changes
	// are name and/or continuation policy, as it means that there
	// are no changes to agentic configuration or task prompt
	configs := []config.Task{
		{Name: "task-1", ContinuePolicy: config.ContinuePolicyDefault},
		{Name: "task-2", ContinuePolicy: config.ContinuePolicyResume},
		{Name: "task-3", ContinuePolicy: config.ContinuePolicyResume},
	}

	var hash string
	for _, cfg := range configs {
		hashed := FromConfig(&cfg).hash
		if hash == "" {
			hash = hashed
		} else {
			assert.Equal(t, hash, hashed)
		}
	}
}

func TestSentinel_DifferentHash(t *testing.T) {
	t.Parallel()

	configs := []config.Task{
		{
			Name:           "task-1",
			ContinuePolicy: config.ContinuePolicyDefault,
		},
		{
			Name:           "task-1",
			ContinuePolicy: config.ContinuePolicyResume,
			Agent:          config.Agent{Model: "model-1"},
		},
		{
			Name:           "task-1",
			ContinuePolicy: config.ContinuePolicyResume,
			Agent:          config.Agent{Model: "model-2"},
			Source:         config.Source{PromptFile: "prompt.md"},
			Description:    "a different task",
		},
	}

	var hash string
	for _, cfg := range configs {
		hashed := FromConfig(&cfg).hash
		if hash == "" {
			hash = hashed
		} else {
			assert.NotEqual(t, hash, hashed)
		}
	}
}

func TestSentinel_Equal(t *testing.T) {
	t.Parallel()
	cfg1 := config.Task{
		Name:           "task-1",
		Description:    "a test task",
		Source:         config.Source{LocalPath: "/source"},
		ContinuePolicy: config.ContinuePolicyResume,
	}
	cfg2 := config.Task{
		Name:           "task-2",
		Description:    "a different test task",
		Source:         config.Source{LocalPath: "/source"},
		ContinuePolicy: config.ContinuePolicyDefault,
	}

	sen1 := FromConfig(&cfg1)
	sen2 := FromConfig(&cfg2)

	assert.Condition(t, Equal(sen1, sen1))
	assert.Condition(t, Equal(sen2, sen2))
	assert.Condition(t, !Equal(sen1, sen2))
}

func TestSentinel_Match(t *testing.T) {
	t.Parallel()
	cfg := config.Task{
		Name:           "task-1",
		Description:    "a test task",
		Source:         config.Source{LocalPath: "/source"},
		ContinuePolicy: config.ContinuePolicyResume,
	}
	sen := FromConfig(&cfg)

	assert.Condition(t, sen.Match(&cfg))

	cfg.Agent.Model = "different-model"
	assert.Condition(t, !sen.Match(&cfg))
}

func TestSentinel_Write(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	cfg := config.Task{
		Name:           "task-1",
		Description:    "a test task",
		Source:         config.Source{LocalPath: "/source"},
		ContinuePolicy: config.ContinuePolicyResume,
	}
	sen := FromConfig(&cfg)

	err := sen.Write(baseDir)

	assert.NotErr(t, err)
	written := testutil.ReadFile(t, sentinelPath(baseDir))
	assert.Equal(t, sen.hash, written)
}
