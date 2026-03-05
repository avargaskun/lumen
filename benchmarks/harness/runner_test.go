package harness

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ory/lumen/benchmarks/tasks"
)

func TestBuildClaudeArgs_Baseline(t *testing.T) {
	task := tasks.Task{
		ProblemStatement: "fix the bug",
		MaxTurns:         20,
		MaxBudgetUSD:     1.50,
	}
	opts := RunOpts{Model: "claude-sonnet-4-6"}

	args := buildClaudeArgs(task, tasks.ScenarioBaseline, opts, "/tmp/mcp.json")

	assertContains(t, args, "--output-format", "stream-json")
	assertContains(t, args, "--strict-mcp-config")
	assertContains(t, args, "--mcp-config", "/tmp/mcp.json")
	assertContains(t, args, "--max-turns", "20")
	assertContains(t, args, "--max-budget-usd", "1.50")
	assertContains(t, args, "--model", "claude-sonnet-4-6")

	// Baseline should NOT have --tools or --allowedTools.
	assertNotContains(t, args, "--tools")
	assertNotContains(t, args, "--allowedTools")
}

func TestBuildClaudeArgs_MCPOnly(t *testing.T) {
	task := tasks.Task{
		ProblemStatement: "fix the bug",
		MaxTurns:         30,
		MaxBudgetUSD:     2.00,
	}
	opts := RunOpts{Model: "claude-sonnet-4-6"}

	args := buildClaudeArgs(task, tasks.ScenarioMCPOnly, opts, "/tmp/mcp.json")

	assertContains(t, args, "--tools", "")
	assertContains(t, args, "--allowedTools", "mcp__lumen__semantic_search,mcp__lumen__index_status")
}

func TestBuildClaudeArgs_MCPFull(t *testing.T) {
	task := tasks.Task{
		ProblemStatement: "fix the bug",
		MaxTurns:         30,
		MaxBudgetUSD:     2.00,
	}
	opts := RunOpts{Model: "claude-sonnet-4-6"}

	args := buildClaudeArgs(task, tasks.ScenarioMCPFull, opts, "/tmp/mcp.json")

	assertContains(t, args, "--allowedTools", "mcp__lumen__semantic_search,mcp__lumen__index_status")

	// mcp-full should NOT have --tools "" (should keep default tools).
	for i, a := range args {
		if a == "--tools" && i+1 < len(args) && args[i+1] == "" {
			t.Error("mcp-full should not have --tools \"\" (disables default tools)")
		}
	}
}

func TestWriteMCPConfig_Baseline(t *testing.T) {
	opts := RunOpts{
		LumenBinary:  "./lumen",
		EmbedModel:   "test-model",
		EmbedBackend: "ollama",
	}

	path, err := writeMCPConfig(opts, tasks.ScenarioBaseline)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var cfg mcpConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}

	if len(cfg.MCPServers) != 0 {
		t.Errorf("baseline should have empty mcpServers, got %d", len(cfg.MCPServers))
	}
}

func TestWriteMCPConfig_MCPOnly(t *testing.T) {
	opts := RunOpts{
		LumenBinary:  "/usr/bin/lumen",
		EmbedModel:   "jina-v2",
		EmbedBackend: "ollama",
	}

	path, err := writeMCPConfig(opts, tasks.ScenarioMCPOnly)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var cfg mcpConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}

	lumen, ok := cfg.MCPServers["lumen"]
	if !ok {
		t.Fatal("expected lumen server in config")
	}
	if lumen.Command != "/usr/bin/lumen" {
		t.Errorf("command = %q, want /usr/bin/lumen", lumen.Command)
	}
	if lumen.Args[0] != "stdio" {
		t.Errorf("args[0] = %q, want stdio", lumen.Args[0])
	}
	if lumen.Env["LUMEN_BACKEND"] != "ollama" {
		t.Errorf("LUMEN_BACKEND = %q", lumen.Env["LUMEN_BACKEND"])
	}
	if lumen.Env["LUMEN_EMBED_MODEL"] != "jina-v2" {
		t.Errorf("LUMEN_EMBED_MODEL = %q", lumen.Env["LUMEN_EMBED_MODEL"])
	}
}

func assertContains(t *testing.T, args []string, vals ...string) {
	t.Helper()
	if len(vals) == 1 {
		for _, a := range args {
			if a == vals[0] {
				return
			}
		}
		t.Errorf("args missing %q: %v", vals[0], args)
		return
	}

	// Check key-value pair.
	for i := 0; i < len(args)-1; i++ {
		if args[i] == vals[0] && args[i+1] == vals[1] {
			return
		}
	}
	t.Errorf("args missing %q %q: %v", vals[0], vals[1], args)
}

func assertNotContains(t *testing.T, args []string, val string) {
	t.Helper()
	for _, a := range args {
		if a == val {
			t.Errorf("args should not contain %q: %v", val, args)
			return
		}
	}
}
