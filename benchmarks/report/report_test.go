package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/benchmarks/harness"
	"github.com/ory/lumen/benchmarks/tasks"
)

func TestLoadResults(t *testing.T) {
	dir := t.TempDir()

	results := []harness.RunResult{
		{
			TaskID:   "task-1",
			Scenario: tasks.ScenarioBaseline,
			Metrics:  harness.RunMetrics{CostUSD: 0.05, InputTokens: 1000, OutputTokens: 200, DurationMS: 5000},
			Validation: harness.ValidationResult{
				Success: true,
			},
		},
		{
			TaskID:   "task-1",
			Scenario: tasks.ScenarioMCPFull,
			Metrics:  harness.RunMetrics{CostUSD: 0.03, InputTokens: 600, OutputTokens: 150, DurationMS: 3000},
			Validation: harness.ValidationResult{
				Success: true,
			},
		},
	}

	for _, r := range results {
		data, _ := json.MarshalIndent(r, "", "  ")
		name := r.TaskID + "-" + string(r.Scenario) + "-result.json"
		_ = os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}

	report, err := LoadResults(dir)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}

	if len(report.results) != 1 {
		t.Fatalf("expected 1 task, got %d", len(report.results))
	}

	base, ok := report.results["task-1"][tasks.ScenarioBaseline]
	if !ok {
		t.Fatal("missing baseline result")
	}
	if base.Metrics.CostUSD != 0.05 {
		t.Errorf("baseline cost = %f, want 0.05", base.Metrics.CostUSD)
	}
}

func TestLoadResults_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadResults(dir)
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
}

func TestLoadResults_IgnoresNonResultFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a non-result file.
	_ = os.WriteFile(filepath.Join(dir, "raw.jsonl"), []byte("data"), 0o644)

	_, err := LoadResults(dir)
	if err == nil {
		t.Fatal("expected error when no result files found")
	}
}

func TestGenerateMarkdown(t *testing.T) {
	dir := t.TempDir()

	results := []harness.RunResult{
		{
			TaskID:   "task-1",
			Scenario: tasks.ScenarioBaseline,
			Metrics: harness.RunMetrics{
				CostUSD: 0.10, InputTokens: 5000, OutputTokens: 500,
				DurationMS: 10000, ToolCalls: map[string]int{"Grep": 3},
			},
			Validation: harness.ValidationResult{Success: true},
		},
		{
			TaskID:   "task-1",
			Scenario: tasks.ScenarioMCPFull,
			Metrics: harness.RunMetrics{
				CostUSD: 0.06, InputTokens: 3000, OutputTokens: 300,
				DurationMS: 6000, ToolCalls: map[string]int{"mcp__lumen__semantic_search": 2},
			},
			Validation: harness.ValidationResult{Success: true},
		},
	}

	for _, r := range results {
		data, _ := json.MarshalIndent(r, "", "  ")
		name := r.TaskID + "-" + string(r.Scenario) + "-result.json"
		_ = os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}

	report, err := LoadResults(dir)
	if err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	report.GenerateMarkdown(&buf)
	md := buf.String()

	if !strings.Contains(md, "# Lumen Benchmark Report") {
		t.Error("missing report header")
	}
	if !strings.Contains(md, "## Summary") {
		t.Error("missing summary section")
	}
	if !strings.Contains(md, "task-1") {
		t.Error("missing task-1 in report")
	}
	if !strings.Contains(md, "## Per-Task Details") {
		t.Error("missing per-task section")
	}
}
