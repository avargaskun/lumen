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

package index

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/chunker"
)

func makeTestChunk(symbol string, startLine, endLine int, content string) chunker.Chunk {
	return chunker.Chunk{
		ID:        "original-id-1234",
		FilePath:  "test.go",
		Symbol:    symbol,
		Kind:      "function",
		StartLine: startLine,
		EndLine:   endLine,
		Content:   content,
	}
}

func TestSplitOversizedChunks_UnderLimit(t *testing.T) {
	c := makeTestChunk("SmallFunc", 1, 5, "func SmallFunc() {\n\treturn\n}\n")
	result := splitOversizedChunks([]chunker.Chunk{c}, 2048)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0].ID != c.ID {
		t.Fatalf("expected unchanged chunk, got different ID")
	}
}

func TestSplitOversizedChunks_SplitsLargeChunk(t *testing.T) {
	// Create a chunk with 100 lines, each ~40 chars = ~4000 chars total
	// With maxTokens=200 (800 chars), this should split into ~5 parts
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("    line %d: some code content here\n", i))
	}
	content := strings.Join(lines, "")
	c := makeTestChunk("BigFunc", 10, 109, content)

	result := splitOversizedChunks([]chunker.Chunk{c}, 200)
	if len(result) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(result))
	}

	// Check symbol format
	for i, r := range result {
		expected := fmt.Sprintf("BigFunc[%d/%d]", i+1, len(result))
		if r.Symbol != expected {
			t.Errorf("chunk %d: expected symbol %q, got %q", i, expected, r.Symbol)
		}
		if r.Kind != "function" {
			t.Errorf("chunk %d: expected kind 'function', got %q", i, r.Kind)
		}
		if r.FilePath != "test.go" {
			t.Errorf("chunk %d: expected file 'test.go', got %q", i, r.FilePath)
		}
	}

	// Line ranges are contiguous and cover original range
	if result[0].StartLine != 10 {
		t.Errorf("first chunk should start at line 10, got %d", result[0].StartLine)
	}
	if result[len(result)-1].EndLine != 109 {
		t.Errorf("last chunk should end at line 109, got %d", result[len(result)-1].EndLine)
	}
	for i := 1; i < len(result); i++ {
		if result[i].StartLine != result[i-1].EndLine+1 {
			t.Errorf("gap between chunk %d (end %d) and %d (start %d)",
				i-1, result[i-1].EndLine, i, result[i].StartLine)
		}
	}

	// IDs are unique
	seen := map[string]bool{}
	for _, r := range result {
		if seen[r.ID] {
			t.Errorf("duplicate ID: %s", r.ID)
		}
		seen[r.ID] = true
	}

	// Content reconstructs to original (stripping prepended signature from parts 2..N)
	firstLine := strings.SplitN(content, "\n", 2)[0]
	signaturePrefix := firstLine + " // ...\n"
	var reconstructed string
	for i, r := range result {
		c := r.Content
		if i > 0 {
			c = strings.TrimPrefix(c, signaturePrefix)
		}
		reconstructed += c
	}
	if reconstructed != content {
		t.Error("reconstructed content does not match original")
	}
}

func TestSplitOversizedChunks_SingleHugeLine(t *testing.T) {
	// One line exceeding maxChars — should pass through as one chunk (no infinite loop)
	content := strings.Repeat("x", 10000) + "\n"
	c := makeTestChunk("HugeLine", 1, 1, content)

	result := splitOversizedChunks([]chunker.Chunk{c}, 100)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk for single huge line, got %d", len(result))
	}
}

func TestSplitOversizedChunks_ZeroMaxTokens(t *testing.T) {
	c := makeTestChunk("Func", 1, 5, "content\n")
	result := splitOversizedChunks([]chunker.Chunk{c}, 0)
	if len(result) != 1 {
		t.Fatalf("expected passthrough with maxTokens=0, got %d chunks", len(result))
	}
}

func TestSplitOversizedChunks_TypeUsesFullBudget(t *testing.T) {
	// With the old code, type chunks got half the budget (typeMaxChars).
	// Now types use the full budget, so a type chunk of ~600 chars should
	// pass through unchanged with maxTokens=200 (800 char budget).
	var lines []string
	for i := 0; i < 15; i++ {
		lines = append(lines, fmt.Sprintf("    field%d: some type here\n", i))
	}
	content := strings.Join(lines, "") // ~38 chars * 15 = ~570 chars
	c := chunker.Chunk{
		ID:        "type-id",
		FilePath:  "test.java",
		Symbol:    "MyClass",
		Kind:      "type",
		StartLine: 1,
		EndLine:   15,
		Content:   content,
	}

	result := splitOversizedChunks([]chunker.Chunk{c}, 200)
	if len(result) != 1 {
		t.Fatalf("expected type chunk to pass through unchanged at full budget, got %d chunks", len(result))
	}
}

func TestPartitionByBlankLines_SplitsAtBlankLines(t *testing.T) {
	// Build content with two method-like sections separated by a blank line.
	// Each section is ~200 chars; maxChars=300 so they can't fit together.
	section1 := make([]string, 5)
	section2 := make([]string, 5)
	for i := range section1 {
		section1[i] = fmt.Sprintf("    line %d: some code content here and more\n", i)
	}
	section1 = append(section1, "\n") // blank line terminator
	for i := range section2 {
		section2[i] = fmt.Sprintf("    line %d: some code content here and more\n", i+10)
	}

	lines := append(section1, section2...)
	parts := partitionByBlankLines(lines, 300)

	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts from blank-line split, got %d", len(parts))
	}

	// Verify first part ends with the blank line.
	firstPart := parts[0]
	lastLine := firstPart[len(firstPart)-1]
	if strings.TrimRight(lastLine, " \t\r\n") != "" {
		t.Errorf("first part should end with blank line, got %q", lastLine)
	}
}

func TestPartitionByBlankLines_FitsInSinglePart(t *testing.T) {
	// Small content with blank lines should remain one part.
	lines := []string{
		"line 1\n",
		"\n",
		"line 3\n",
	}
	parts := partitionByBlankLines(lines, 10000)
	if len(parts) != 1 {
		t.Fatalf("expected 1 part when content fits budget, got %d", len(parts))
	}
}

func TestPartitionByBlankLines_OversizedSectionFallsBack(t *testing.T) {
	// A single section larger than maxChars should fall back to line-based splitting.
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("    line %d: some code content here\n", i))
	}
	// No blank lines — one giant section.
	parts := partitionByBlankLines(lines, 200)
	if len(parts) < 2 {
		t.Fatalf("expected fallback line-split for oversized section, got %d parts", len(parts))
	}
}

func TestSplitOversizedChunks_TypeSplitsAtBlankLines(t *testing.T) {
	// Type chunk with method sections separated by blank lines — should split at blank lines.
	var content string
	for i := 0; i < 3; i++ {
		for j := 0; j < 8; j++ {
			content += fmt.Sprintf("    line %d-%d: some code content here and more text\n", i, j)
		}
		content += "\n" // blank line between sections
	}

	c := chunker.Chunk{
		ID:        "type-id",
		FilePath:  "Owner.java",
		Symbol:    "Owner",
		Kind:      "type",
		StartLine: 1,
		EndLine:   27,
		Content:   content,
	}

	// Budget: 300 chars — each section (~360 chars) exceeds budget individually,
	// so it falls back to line splitting per section.
	result := splitOversizedChunks([]chunker.Chunk{c}, 75)
	if len(result) < 2 {
		t.Fatalf("expected type chunk to split, got %d chunks", len(result))
	}
	// Check parts carry "type" kind through.
	for _, r := range result {
		if r.Kind != "type" {
			t.Errorf("split chunk should keep kind=type, got %q", r.Kind)
		}
	}
}

func TestSplitOversizedChunks_MixedSizes(t *testing.T) {
	small := makeTestChunk("Small", 1, 3, "small\n")
	var bigLines []string
	for i := 0; i < 50; i++ {
		bigLines = append(bigLines, fmt.Sprintf("line %d content here\n", i))
	}
	big := makeTestChunk("Big", 10, 59, strings.Join(bigLines, ""))

	result := splitOversizedChunks([]chunker.Chunk{small, big}, 100)
	// First chunk should be the small one unchanged
	if result[0].Symbol != "Small" {
		t.Errorf("expected first chunk to be Small, got %s", result[0].Symbol)
	}
	// Remaining chunks should be splits of Big
	if len(result) < 3 {
		t.Fatalf("expected at least 3 chunks (1 small + 2+ splits), got %d", len(result))
	}
}
