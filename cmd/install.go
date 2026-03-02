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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aeneasr/lumen/internal/config"
	"github.com/aeneasr/lumen/internal/embedder"
	"github.com/spf13/cobra"
)

func init() {
	defaultName := filepath.Base(os.Args[0])
	installCmd.Flags().StringP("mcp-name", "n", defaultName, "name to register with claude mcp add")
	installCmd.Flags().StringP("model", "m", "", "skip interactive model selection, use this model")
	installCmd.Flags().StringP("file", "f", "", "target CLAUDE.md/agents.md path (auto-detected if omitted)")
	installCmd.Flags().Bool("dry-run", false, "print actions without executing them")
	installCmd.Flags().Bool("no-mcp", false, "skip MCP registration, only write the CLAUDE.md snippet")
	installCmd.Flags().Bool("no-claude-md", false, "skip CLAUDE.md/agents.md update, only register MCP")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install [project-path]",
	Short: "Install lumen MCP server and configure code search directives",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	projectPath, err := resolveProjectPath(args)
	if err != nil {
		return err
	}

	mcpName, _ := cmd.Flags().GetString("mcp-name")
	modelFlag, _ := cmd.Flags().GetString("model")
	fileFlag, _ := cmd.Flags().GetString("file")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noMCP, _ := cmd.Flags().GetBool("no-mcp")
	noClaudeMD, _ := cmd.Flags().GetBool("no-claude-md")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Phase 1: Service detection
	backend, host, err := detectAndSelectService(ctx)
	if err != nil {
		return err
	}

	// Phase 2: Model selection
	selectedModel, err := selectModel(ctx, backend, host, modelFlag)
	if err != nil {
		return err
	}

	// Phase 3: MCP registration
	if !noMCP {
		if err := registerMCP(mcpName, backend, selectedModel, dryRun); err != nil {
			return err
		}
	}

	// Phase 4: CLAUDE.md / agents.md upsert
	if !noClaudeMD {
		if err := upsertClaudeMD(projectPath, fileFlag, mcpName, dryRun); err != nil {
			return err
		}
	}

	return nil
}

func resolveProjectPath(args []string) (string, error) {
	if len(args) > 0 {
		return filepath.Abs(args[0])
	}
	return os.Getwd()
}

// --- Phase 1: Service detection ---

func detectServices(ctx context.Context) (ollamaOK, lmstudioOK bool) {
	ollamaHost := config.EnvOrDefault("OLLAMA_HOST", "http://localhost:11434")
	lmstudioHost := config.EnvOrDefault("LM_STUDIO_HOST", "http://localhost:1234")

	type result struct {
		name string
		ok   bool
	}

	ch := make(chan result, 2)
	go func() { ch <- result{"ollama", probeService(ctx, ollamaHost+"/api/tags")} }()
	go func() { ch <- result{"lmstudio", probeService(ctx, lmstudioHost+"/v1/models")} }()

	for range 2 {
		r := <-ch
		switch r.name {
		case "ollama":
			ollamaOK = r.ok
		case "lmstudio":
			lmstudioOK = r.ok
		}
	}

	return ollamaOK, lmstudioOK
}

func probeService(ctx context.Context, url string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}

func detectAndSelectService(ctx context.Context) (backend, host string, err error) {
	ollamaHost := config.EnvOrDefault("OLLAMA_HOST", "http://localhost:11434")
	lmstudioHost := config.EnvOrDefault("LM_STUDIO_HOST", "http://localhost:1234")

	fmt.Fprintln(os.Stderr, "Detecting embedding services...")

	ollamaOK, lmstudioOK := detectServices(ctx)

	printServiceStatus("Ollama", ollamaHost, ollamaOK)
	printServiceStatus("LM Studio", lmstudioHost, lmstudioOK)

	switch {
	case !ollamaOK && !lmstudioOK:
		return "", "", fmt.Errorf(
			"no embedding service detected\n" +
				"  Install Ollama:    https://ollama.com\n" +
				"  Install LM Studio: https://lmstudio.ai",
		)
	case ollamaOK && !lmstudioOK:
		return config.BackendOllama, ollamaHost, nil
	case !ollamaOK && lmstudioOK:
		return config.BackendLMStudio, lmstudioHost, nil
	default:
		// Both available: prompt
		return promptServiceSelection(ollamaHost, lmstudioHost)
	}
}

func printServiceStatus(name, host string, ok bool) {
	trimHost := strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")
	if ok {
		fmt.Fprintf(os.Stderr, "  \u2713 %-12s (%s)\n", name, trimHost)
	} else {
		fmt.Fprintf(os.Stderr, "  \u2717 %-12s (%s \u2014 not running)\n", name, trimHost)
	}
}

func promptServiceSelection(ollamaHost, lmstudioHost string) (backend, host string, err error) {
	if !stdinIsTTY() {
		return "", "", fmt.Errorf("stdin is not a terminal — use OLLAMA_HOST or LM_STUDIO_HOST env vars to disambiguate")
	}

	fmt.Fprintln(os.Stderr, "\nBoth services are available. Which backend should be used?")
	fmt.Fprintln(os.Stderr, "  1. Ollama")
	fmt.Fprintln(os.Stderr, "  2. LM Studio")
	fmt.Fprint(os.Stderr, "Pick a service [1]: ")

	line, err := readLine()
	if err != nil {
		return "", "", fmt.Errorf("read input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" || line == "1" {
		return config.BackendOllama, ollamaHost, nil
	}
	if line == "2" {
		return config.BackendLMStudio, lmstudioHost, nil
	}
	return "", "", fmt.Errorf("invalid selection %q: enter 1 or 2", line)
}

// --- Phase 2: Model selection ---

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type lmstudioModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func fetchJSON[T any](ctx context.Context, url string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer func() { _ = resp.Body.Close() }()

	var data T
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return zero, err
	}
	return data, nil
}

func fetchOllamaModels(ctx context.Context, host string) ([]string, error) {
	data, err := fetchJSON[ollamaTagsResponse](ctx, host+"/api/tags")
	if err != nil {
		return nil, err
	}
	names := make([]string, len(data.Models))
	for i, m := range data.Models {
		names[i] = m.Name
	}
	return names, nil
}

func fetchLMStudioModels(ctx context.Context, host string) ([]string, error) {
	data, err := fetchJSON[lmstudioModelsResponse](ctx, host+"/v1/models")
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(data.Data))
	for i, m := range data.Data {
		ids[i] = m.ID
	}
	return ids, nil
}

func selectModel(ctx context.Context, backend, host, modelFlag string) (string, error) {
	var models []string
	var err error

	if backend == config.BackendOllama {
		models, err = fetchOllamaModels(ctx, host)
	} else {
		models, err = fetchLMStudioModels(ctx, host)
	}
	if err != nil {
		return "", fmt.Errorf("list models: %w", err)
	}

	if modelFlag != "" {
		// Validate the model exists (warn but don't fail if unknown)
		found := false
		for _, m := range models {
			if m == modelFlag {
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Warning: model %q not found in service response\n", modelFlag)
		}
		return modelFlag, nil
	}

	if len(models) == 0 {
		return "", fmt.Errorf("no models available in %s — pull a model first", backend)
	}

	return promptModelSelection(models, backend)
}

func promptModelSelection(models []string, backend string) (string, error) {
	if !stdinIsTTY() {
		return "", fmt.Errorf("stdin is not a terminal — use --model to specify a model non-interactively")
	}

	// Sort: known models first, then unknowns alphabetically
	slices.SortFunc(models, func(a, b string) int {
		_, aKnown := embedder.KnownModels[a]
		_, bKnown := embedder.KnownModels[b]
		if aKnown != bKnown {
			if aKnown {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})

	backendLabel := "Ollama"
	if backend == config.BackendLMStudio {
		backendLabel = "LM Studio"
	}

	fmt.Fprintf(os.Stderr, "\nAvailable models (%s):\n", backendLabel)
	for i, name := range models {
		spec, known := embedder.KnownModels[name]
		if known {
			recommended := ""
			if name == embedder.DefaultOllamaModel && backend == config.BackendOllama {
				recommended = "  [recommended]"
			} else if name == embedder.DefaultLMStudioModel && backend == config.BackendLMStudio {
				recommended = "  [recommended]"
			}
			fmt.Fprintf(os.Stderr, "  %d. %-45s %4d dims  %5d ctx  %s%s\n",
				i+1, name, spec.Dims, spec.CtxLength, spec.SizeHint, recommended)
		} else {
			fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, name)
		}
	}

	fmt.Fprint(os.Stderr, "\nPick a model [1]: ")

	line, err := readLine()
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return models[0], nil
	}

	// Try numeric selection
	if idx, err := strconv.Atoi(line); err == nil {
		if idx < 1 || idx > len(models) {
			return "", fmt.Errorf("invalid selection %d: enter 1-%d", idx, len(models))
		}
		return models[idx-1], nil
	}

	// Try model name directly
	for _, m := range models {
		if m == line {
			return m, nil
		}
	}
	return "", fmt.Errorf("invalid selection %q", line)
}

// --- Phase 3: MCP registration ---

func registerMCP(mcpName, backend, model string, dryRun bool) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	fmt.Fprintln(os.Stderr, "\nRegistering MCP server...")

	claudeErr := registerClaudeCode(mcpName, binaryPath, backend, model, dryRun)
	codexErr := registerCodex(mcpName, binaryPath, backend, model, dryRun)

	if claudeErr != nil && !isNotFound(claudeErr) {
		fmt.Fprintf(os.Stderr, "  Warning: claude registration failed: %v\n", claudeErr)
	}
	if codexErr != nil && !isNotFound(codexErr) {
		fmt.Fprintf(os.Stderr, "  Warning: codex registration failed: %v\n", codexErr)
	}

	return nil
}

func registerClaudeCode(mcpName, binaryPath, backend, model string, dryRun bool) error {
	args := []string{
		"mcp", "add",
		"--scope", "user",
		"-e", "LUMEN_BACKEND=" + backend,
		"-e", "LUMEN_EMBED_MODEL=" + model,
		mcpName, binaryPath, "--", "stdio",
	}

	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Fprintf(os.Stderr, "  ! Claude Code  (claude not in PATH — skipping)\n")
		return err
	}

	cmdStr := "claude " + strings.Join(args, " ")
	if dryRun {
		fmt.Fprintf(os.Stderr, "  [dry-run] %s\n", cmdStr)
		return nil
	}

	out, err := exec.Command("claude", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", cmdStr, strings.TrimSpace(string(out)))
	}
	fmt.Fprintf(os.Stderr, "  \u2713 Claude Code  (%s)\n", cmdStr)
	return nil
}

func registerCodex(mcpName, binaryPath, backend, model string, dryRun bool) error {
	args := []string{
		"mcp", "add",
		"--env", "LUMEN_BACKEND=" + backend,
		"--env", "LUMEN_EMBED_MODEL=" + model,
		binaryPath, "stdio", mcpName,
	}

	if _, err := exec.LookPath("codex"); err != nil {
		// Codex not in PATH: skip silently
		return err
	}

	cmdStr := "codex " + strings.Join(args, " ")
	if dryRun {
		fmt.Fprintf(os.Stderr, "  [dry-run] %s\n", cmdStr)
		return nil
	}

	out, err := exec.Command("codex", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", cmdStr, strings.TrimSpace(string(out)))
	}
	fmt.Fprintf(os.Stderr, "  \u2713 Codex        (%s)\n", cmdStr)
	return nil
}

func isNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}

// --- Phase 4: CLAUDE.md / agents.md upsert ---

func upsertClaudeMD(projectPath, fileFlag, mcpName string, dryRun bool) error {
	targetFile, err := resolveTargetFile(projectPath, fileFlag)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nUpdating %s...\n", filepath.Base(targetFile))

	existing := ""
	if data, readErr := os.ReadFile(targetFile); readErr == nil {
		existing = string(data)
	}

	snippet := generateSnippet(mcpName)
	updated := upsertSnippet(existing, snippet)

	rel, err := filepath.Rel(projectPath, targetFile)
	if err != nil {
		rel = targetFile
	}

	if dryRun {
		fmt.Fprintf(os.Stderr, "  [dry-run] Would write snippet to ./%s (mcp-name: %s)\n", rel, mcpName)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	if err := os.WriteFile(targetFile, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  \u2713 Wrote snippet to ./%s (mcp-name: %s)\n", rel, mcpName)
	return nil
}

// resolveTargetFile determines which file to upsert the snippet into.
func resolveTargetFile(projectPath, fileFlag string) (string, error) {
	if fileFlag != "" {
		if filepath.IsAbs(fileFlag) {
			return fileFlag, nil
		}
		return filepath.Join(projectPath, fileFlag), nil
	}

	candidates := []struct {
		path      string
		checkOnly bool // true = only check if redirect; false = check content for @agents.md
	}{
		{filepath.Join(projectPath, "CLAUDE.md"), false},
		{filepath.Join(projectPath, "agents.md"), true},
		{filepath.Join(projectPath, ".claude", "CLAUDE.md"), false},
		{filepath.Join(projectPath, ".claude", "agents.md"), true},
	}

	for _, c := range candidates {
		if c.checkOnly {
			if fileExists(c.path) {
				return c.path, nil
			}
			continue
		}
		// CLAUDE.md: check if it redirects to agents.md
		if data, err := os.ReadFile(c.path); err == nil {
			content := strings.TrimSpace(string(data))
			if content == "@agents.md" {
				// Write to agents.md in the same directory
				return filepath.Join(filepath.Dir(c.path), "agents.md"), nil
			}
			// Regular CLAUDE.md with content
			return c.path, nil
		}
	}

	// Nothing found — create CLAUDE.md at project root
	return filepath.Join(projectPath, "CLAUDE.md"), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const (
	snippetStart = "<!-- lumen:start -->"
	snippetEnd   = "<!-- lumen:end -->"
)

// generateSnippet returns the CLAUDE.md code search directive for the given MCP server name.
func generateSnippet(mcpName string) string {
	return snippetStart + "\n" +
		"# Code Search\n\n" +
		"ALWAYS use `mcp__" + mcpName + "__semantic_search` as the FIRST tool for code discovery and exploration.\n" +
		"Do NOT default to Grep, Glob, or Read for search tasks — only use them for exact literal string lookups.\n\n" +
		"Before using Grep, Glob, Find, or Read for any search, stop and ask: \"Do I already know the exact\n" +
		"literal string I'm searching for?\" If not, use `mcp__" + mcpName + "__semantic_search`. If semantic\n" +
		"search is unavailable, Grep/Glob are acceptable fallbacks.\n" +
		snippetEnd
}

// upsertSnippet inserts or replaces the snippet markers in existing content.
// It is a pure function and is directly testable.
func upsertSnippet(existing, snippet string) string {
	startIdx := strings.Index(existing, snippetStart)
	endIdx := strings.Index(existing, snippetEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Replace between markers (inclusive)
		return existing[:startIdx] + snippet + existing[endIdx+len(snippetEnd):]
	}

	// Append
	if strings.TrimSpace(existing) == "" {
		return snippet + "\n"
	}
	// Ensure blank line separator before appending
	if !strings.HasSuffix(existing, "\n\n") {
		if strings.HasSuffix(existing, "\n") {
			existing += "\n"
		} else {
			existing += "\n\n"
		}
	}
	return existing + snippet + "\n"
}

// --- Helpers ---

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func readLine() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}
