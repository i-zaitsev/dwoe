package workspace

import (
	"slices"
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/config/provider"
	"github.com/i-zaitsev/dwoe/internal/docker"
	"github.com/i-zaitsev/dwoe/internal/state"
)

func newWorkspace(name provider.ModelProviderName) *Workspace {
	cfg := config.NewTask("ws-name", func(task *config.Task) {
		task.NoProxy = true
		task.Agent = &config.Agent{Provider: provider.Provider{Name: name, Model: "test-model"}}
	}, config.WithGlobals(config.NewGlobal()), config.WithDefaults())
	return &Workspace{
		Workspace: state.EmptyWorkspace("ws-id", "ws-name"),
		Config:    cfg,
	}
}

func TestWorkspace_Env_Provider(t *testing.T) {
	tests := []struct {
		name     string
		provider provider.ModelProviderName
		wantName string
	}{
		{"unknown defaults to anthropic", provider.ModelProviderUnknown, "anthropic"},
		{"anthropic", provider.ModelProviderAnthropic, "anthropic"},
		{"openai", provider.ModelProviderOpenAI, "openai"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := strings.Join(newWorkspace(tt.provider).Env(), "\n")
			assert.Contains(t, env, "AGENT_PROVIDER="+tt.wantName)
			assert.Contains(t, env, "AGENT_MODEL=test-model")
			assert.Contains(t, env, "CLAUDE_MODEL=test-model")
		})
	}
}

func TestWorkspace_Env_OpenAIAuthPassthrough(t *testing.T) {
	t.Setenv("CODEX_API_KEY", "sk-codex-test")
	t.Setenv("OPENAI_API_KEY", "sk-test")

	env := strings.Join(newWorkspace(provider.ModelProviderOpenAI).Env(), "\n")

	assert.Contains(t, env, "CODEX_API_KEY=sk-codex-test")
	assert.Contains(t, env, "OPENAI_API_KEY=sk-test")
}

func TestWorkspace_Mounts_SettingsProviderSpecific(t *testing.T) {
	t.Parallel()
	const settings = "/home/agent/.claude/settings.json"

	claudeTargets := mountTargets(newWorkspace(provider.ModelProviderAnthropic).Mounts())
	assert.Equal(t, slices.Contains(claudeTargets, settings), true)

	openaiTargets := mountTargets(newWorkspace(provider.ModelProviderOpenAI).Mounts())
	assert.Equal(t, slices.Contains(openaiTargets, settings), false)
}

func mountTargets(mounts []docker.Mount) []string {
	targets := make([]string, len(mounts))
	for i, m := range mounts {
		targets[i] = m.Target
	}
	return targets
}

func TestWorkspace_Env_PromptFileSynthesizesTaskPrompt(t *testing.T) {
	t.Parallel()
	testFile := "testdata/prompt.txt"
	tests := []struct {
		name       string
		taskPrompt string
		promptFile string
		wantEnv    string
	}{
		{
			name:       "prompt_file",
			taskPrompt: "",
			promptFile: testFile,
			wantEnv:    "TASK_PROMPT=Follow the instructions in " + testFile,
		},
		{
			name:       "prompt_file_and_task_prompt",
			taskPrompt: "do the thing",
			promptFile: testFile,
			wantEnv:    "TASK_PROMPT=do the thing",
		},
		{
			name:       "task_prompt",
			taskPrompt: "do the thing",
			promptFile: "",
			wantEnv:    "TASK_PROMPT=do the thing",
		},
		{
			name:       "no_prompt",
			taskPrompt: "",
			promptFile: "",
			wantEnv:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ws := &Workspace{
				Workspace: state.EmptyWorkspace("ws-id", "ws-name"),
				Config: config.NewTask("ws-name", func(task *config.Task) {
					task.NoProxy = true
					task.Agent = &config.Agent{TaskPrompt: tt.taskPrompt}
					task.Source = config.Source{PromptFile: tt.promptFile}
				}, config.WithGlobals(config.NewGlobal()), config.WithDefaults()),
			}
			env := strings.Join(ws.Env(), "\n")
			if tt.wantEnv == "" {
				assert.Condition(t, !strings.Contains(env, "TASK_PROMPT="))
			} else {
				assert.Contains(t, env, tt.wantEnv)
			}
		})
	}
}
