#!/usr/bin/env bash
# setup_repo.sh — Clone a repo at a pinned commit into the workspace.
#
# Usage: setup_repo.sh <clone_url> <commit_or_tag> <dest_dir> [setup_commands...]
#
# Idempotent: skips clone if dest_dir already exists at the correct commit.

set -euo pipefail

CLONE_URL="$1"
COMMIT="$2"
DEST_DIR="$3"
shift 3
SETUP_CMDS=("$@")

if [[ -d "$DEST_DIR/.git" ]]; then
    CURRENT=$(git -C "$DEST_DIR" rev-parse HEAD 2>/dev/null || echo "")
    TARGET=$(git -C "$DEST_DIR" rev-parse "$COMMIT" 2>/dev/null || echo "unknown")
    if [[ "$CURRENT" == "$TARGET" ]]; then
        echo "[setup] Repo already at $COMMIT, skipping clone"
        exit 0
    fi
    echo "[setup] Repo exists but at wrong commit, resetting..."
    git -C "$DEST_DIR" fetch --depth=1 origin "$COMMIT"
    git -C "$DEST_DIR" checkout "$COMMIT" --force
else
    echo "[setup] Cloning $CLONE_URL at $COMMIT -> $DEST_DIR"
    mkdir -p "$(dirname "$DEST_DIR")"
    git clone --depth=1 --branch "$COMMIT" "$CLONE_URL" "$DEST_DIR" 2>/dev/null \
        || git clone "$CLONE_URL" "$DEST_DIR" && git -C "$DEST_DIR" checkout "$COMMIT"
fi

# Clean any leftover state from previous trials
git -C "$DEST_DIR" checkout . 2>/dev/null || true
git -C "$DEST_DIR" clean -fd 2>/dev/null || true

# Run setup commands if any
for cmd in "${SETUP_CMDS[@]}"; do
    if [[ -n "$cmd" ]]; then
        echo "[setup] Running: $cmd"
        (cd "$DEST_DIR" && eval "$cmd")
    fi
done

echo "[setup] Ready: $DEST_DIR"
