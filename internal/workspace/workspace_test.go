package workspace

import (
	"strings"
	"testing"

	"github.com/i-zaitsev/dwoe/internal/assert"
	"github.com/i-zaitsev/dwoe/internal/config"
	"github.com/i-zaitsev/dwoe/internal/state"
)

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
				Config: &config.Task{
					NoProxy: true,
					Agent: config.Agent{
						TaskPrompt: tt.taskPrompt,
					},
					Source: config.Source{
						PromptFile: tt.promptFile,
					},
				},
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
