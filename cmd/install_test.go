// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- upsertSnippet tests ---

func TestUpsertSnippet_EmptyFile(t *testing.T) {
	snippet := generateSnippet("lumen")
	result := upsertSnippet("", snippet)
	if !strings.Contains(result, snippetStart) {
		t.Error("expected snippet start marker")
	}
	if !strings.Contains(result, snippetEnd) {
		t.Error("expected snippet end marker")
	}
	if !strings.Contains(result, "mcp__lumen__semantic_search") {
		t.Error("expected tool name in snippet")
	}
}

func TestUpsertSnippet_AppendToExisting(t *testing.T) {
	existing := "# My Project\n\nSome docs here.\n"
	snippet := generateSnippet("lumen")
	result := upsertSnippet(existing, snippet)

	if !strings.HasPrefix(result, existing) {
		t.Error("existing content should be preserved at the start")
	}
	if !strings.Contains(result, snippetStart) {
		t.Error("expected snippet start marker")
	}
	if !strings.Contains(result, snippetEnd) {
		t.Error("expected snippet end marker")
	}
}

func TestUpsertSnippet_Idempotent(t *testing.T) {
	snippet := generateSnippet("my-server")
	first := upsertSnippet("", snippet)
	second := upsertSnippet(first, snippet)

	if first != second {
		t.Errorf("upsertSnippet is not idempotent:\nfirst=%q\nsecond=%q", first, second)
	}
}

func TestUpsertSnippet_ReplacesExistingMarkers(t *testing.T) {
	original := "# Docs\n\n<!-- lumen:start -->\nold content\n<!-- lumen:end -->\n\n## More\n"
	newSnippet := generateSnippet("new-name")
	result := upsertSnippet(original, newSnippet)

	if strings.Contains(result, "old content") {
		t.Error("old content should be replaced")
	}
	if !strings.Contains(result, "new-name") {
		t.Error("new snippet should contain the new MCP name")
	}
	// Prefix before the markers is preserved
	if !strings.HasPrefix(result, "# Docs\n\n") {
		t.Error("content before markers should be preserved")
	}
	// Suffix after the markers is preserved
	if !strings.Contains(result, "\n\n## More\n") {
		t.Error("content after markers should be preserved")
	}
}

func TestUpsertSnippet_BlankLineSeparator(t *testing.T) {
	// When existing content has no trailing newline, two newlines should be added
	existing := "# My Docs"
	snippet := generateSnippet("lumen")
	result := upsertSnippet(existing, snippet)

	if !strings.Contains(result, "# My Docs\n\n"+snippetStart) {
		t.Errorf("expected double newline before snippet, got: %q", result)
	}
}

// --- generateSnippet tests ---

func TestGenerateSnippet(t *testing.T) {
	cases := []struct {
		name    string
		wantRef string
	}{
		{"lumen", "mcp__lumen__semantic_search"},
		{"my-custom-server", "mcp__my-custom-server__semantic_search"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snippet := generateSnippet(tc.name)
			if !strings.Contains(snippet, tc.wantRef) {
				t.Errorf("expected %q in snippet, got: %s", tc.wantRef, snippet)
			}
			if !strings.HasPrefix(snippet, snippetStart) {
				t.Error("snippet should start with start marker")
			}
			if !strings.HasSuffix(snippet, snippetEnd) {
				t.Error("snippet should end with end marker")
			}
		})
	}
}

// --- resolveTargetFile tests ---

func TestResolveTargetFile_ExplicitFlag(t *testing.T) {
	dir := t.TempDir()
	flagPath := filepath.Join(dir, "custom.md")

	got, err := resolveTargetFile(dir, flagPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != flagPath {
		t.Errorf("got %q, want %q", got, flagPath)
	}
}

func TestResolveTargetFile_ExplicitFlagRelative(t *testing.T) {
	dir := t.TempDir()

	got, err := resolveTargetFile(dir, "custom.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "custom.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveTargetFile_NoneExist(t *testing.T) {
	dir := t.TempDir()

	got, err := resolveTargetFile(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "CLAUDE.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveTargetFile_ClaudeMDExists(t *testing.T) {
	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, []byte("# My Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveTargetFile(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != claudeMD {
		t.Errorf("got %q, want %q", got, claudeMD)
	}
}

func TestResolveTargetFile_ClaudeMDRedirectsToAgentsMD(t *testing.T) {
	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, []byte("@agents.md"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveTargetFile(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "agents.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveTargetFile_ClaudeSubdirClaudeMD(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	claudeMD := filepath.Join(subdir, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, []byte("# Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveTargetFile(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != claudeMD {
		t.Errorf("got %q, want %q", got, claudeMD)
	}
}

func TestResolveTargetFile_AgentsMDFallback(t *testing.T) {
	dir := t.TempDir()
	agentsMD := filepath.Join(dir, "agents.md")
	if err := os.WriteFile(agentsMD, []byte("# Agents\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveTargetFile(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != agentsMD {
		t.Errorf("got %q, want %q", got, agentsMD)
	}
}

func TestResolveTargetFile_ClaudeMDWithNewlineRedirect(t *testing.T) {
	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	// Trailing whitespace/newline should still match
	if err := os.WriteFile(claudeMD, []byte("@agents.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveTargetFile(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "agents.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
