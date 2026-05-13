#!/usr/bin/env bash
set -euo pipefail

# Devcontainer Post-Create-Hook (go Template).
# Idempotent — wird bei jeder Container-Erstellung ausgefuehrt.

echo "[post-create] Fixing named-volume permissions..."
# Docker-Named-Volumes werden mit root:root erstellt; der vscode-User braucht Schreibzugriff.
sudo chown -R vscode:vscode /go/pkg/mod 2>/dev/null || true
# Wichtig: Der Volume-Mount auf /home/vscode/.cache/go-build resetet auch die Eigentuemerschaft
# des Parent-/home/vscode/.cache auf root, sonst kann z.B. golangci-lint seinen Subdir nicht anlegen.
sudo chown vscode:vscode /home/vscode/.cache 2>/dev/null || true
sudo chown -R vscode:vscode /home/vscode/.cache/go-build 2>/dev/null || true

# Optional: Wenn das Repo zusaetzlich ein eingebettetes Node-Frontend hat (z.B. web/node_modules
# als drittes Named-Volume — siehe lens-Pilot): hier ergaenzen.
# sudo chown -R vscode:vscode "/workspaces/${PWD##*/}/web/node_modules" 2>/dev/null || true

echo "[post-create] Versions:"
echo "  $(go version)"
echo "  Delve:   $(dlv version 2>&1 | head -1)"

echo "[post-create] Activating pre-push hook (core.hooksPath=.githooks)..."
if git rev-parse --git-dir >/dev/null 2>&1; then
    git config core.hooksPath .githooks
    echo "[post-create] core.hooksPath=.githooks set"
else
    echo "[post-create] WARN: git dir not reachable (Worktree-mit-externem-gitdir?)."
    echo "[post-create] WARN: Push-Block-Hook NICHT aktiv. Auf dem Host setzen:"
    echo "[post-create] WARN:   git config core.hooksPath .githooks"
fi
