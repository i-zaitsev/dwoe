#!/usr/bin/env bash
set -euo pipefail

log() { echo "[$(date '+%H:%M:%S')] $*"; }

log "Starting workspace ${WORKSPACE_ID}"

# Git setup
cd /workspace
git config --global --add safe.directory /workspace
git config --global init.defaultBranch main
git config --global user.email "${GIT_USER_EMAIL}"
git config --global user.name "${GIT_USER_NAME}"

if [ ! -d ".git" ]; then
    git init
    git add .
    git commit -m "Initial commit" || true
fi

# Run the agent. On success (exit 0), the container exits.
# On failure, retry with exponential backoff:
# 16s, 32s, 64s, 128s, 256s, 512s, 1024s (cap ~17 min).

delay=16
max_delay=1024
attempt=0
task_prompt="${TASK_PROMPT:-Read the source code, task files, and CLAUDE.md in the current directory. Complete the task described in the task files.}"

while true; do
    attempt=$((attempt + 1))
    log "Agent run #${attempt}"

    set +e
    claude -p "$task_prompt" \
        --model "${CLAUDE_MODEL}" \
        --max-turns "${MAX_TURNS}" \
        --output-format stream-json \
        --verbose
    exit_code=$?
    set -e

    if [ $exit_code -eq 0 ]; then
        log "Agent completed successfully (run #${attempt})"
        exit 0
    fi

    log "Agent exited with code ${exit_code} — retrying in ${delay}s (run #${attempt})"
    sleep $delay
    delay=$((delay * 2))
    if [ $delay -gt $max_delay ]; then
        delay=$max_delay
    fi
done
