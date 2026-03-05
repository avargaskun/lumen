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

package chunker

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// QueryDef defines a tree-sitter query pattern and the chunk kind it produces.
// Pattern must have captures: @decl (full declaration) and @name (identifier).
type QueryDef struct {
	Pattern string
	Kind    string
}

// LanguageDef bundles a tree-sitter language with its query patterns.
type LanguageDef struct {
	Language *sitter.Language
	Queries  []QueryDef
}

type compiledRule struct {
	query *sitter.Query
	kind  string
}

// TreeSitterChunker implements Chunker using tree-sitter.
type TreeSitterChunker struct {
	language *sitter.Language
	rules    []compiledRule
}

// NewTreeSitterChunker compiles the queries in def and returns a TreeSitterChunker.
func NewTreeSitterChunker(def LanguageDef) (*TreeSitterChunker, error) {
	rules := make([]compiledRule, 0, len(def.Queries))
	for _, qd := range def.Queries {
		q, err := sitter.NewQuery([]byte(qd.Pattern), def.Language)
		if err != nil {
			return nil, fmt.Errorf("compile query for kind %q: %w", qd.Kind, err)
		}
		rules = append(rules, compiledRule{
			query: q,
			kind:  qd.Kind,
		})
	}
	return &TreeSitterChunker{language: def.Language, rules: rules}, nil
}

// mustTreeSitterChunker panics if NewTreeSitterChunker returns an error.
// Use only for hardcoded query patterns.
func mustTreeSitterChunker(def LanguageDef) *TreeSitterChunker {
	c, err := NewTreeSitterChunker(def)
	if err != nil {
		panic(fmt.Sprintf("invalid tree-sitter query: %v", err))
	}
	return c
}

// Chunk parses content and returns semantic code chunks.
func (c *TreeSitterChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	root, err := sitter.ParseCtx(context.Background(), content, c.language)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	var chunks []Chunk

	for _, rule := range c.rules {
		qc := sitter.NewQueryCursor()
		qc.Exec(rule.query, root)

		for {
			m, ok := qc.NextMatch()
			if !ok {
				break
			}

			var declNode, nameNode *sitter.Node
			for _, cap := range m.Captures {
				switch rule.query.CaptureNameForId(cap.Index) {
				case "decl":
					declNode = cap.Node
				case "name":
					nameNode = cap.Node
				}
			}

			if declNode == nil || nameNode == nil {
				continue
			}

			startLine := int(declNode.StartPoint().Row) + 1
			endLine := int(declNode.EndPoint().Row) + 1
			snippet := declNode.Content(content)
			symbol := nameNode.Content(content)
			if rule.kind == "method" || rule.kind == "function" {
				if parentNode, className := findEnclosingType(declNode, content); className != "" {
					symbol = className + "." + symbol
					if header := extractClassHeader(parentNode, content); header != "" {
						snippet = header + "\n    // ...\n" + snippet
					}
				}
			}

			chunks = append(chunks, makeChunk(filePath, symbol, rule.kind, startLine, endLine, snippet))
		}
	}

	return chunks, nil
}

// findEnclosingType walks up the AST looking for a class/module/impl container
// and returns both the container node and its source name. Returns nil, "" if
// the node is at module scope.
func findEnclosingType(node *sitter.Node, content []byte) (*sitter.Node, string) {
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.Type() {
		case "class_declaration", "abstract_class_declaration", "class_specifier",
			"class_definition",
			"class", "module",
			"internal_module",
			"trait_item", "struct_item",
			"interface_declaration":
			if nameChild := p.ChildByFieldName("name"); nameChild != nil {
				return p, nameChild.Content(content)
			}
		case "impl_item":
			if typeChild := p.ChildByFieldName("type"); typeChild != nil {
				return p, typeChild.Content(content)
			}
		case "singleton_class":
			if outer := p.Parent(); outer != nil {
				switch outer.Type() {
				case "class", "module":
					if nameChild := outer.ChildByFieldName("name"); nameChild != nil {
						return outer, nameChild.Content(content)
					}
				}
			}
		}
	}
	return nil, ""
}

// extractClassHeader returns the opening declaration line(s) of a type node —
// everything up to and including the first line ending with '{', ':', or ' do',
// capped at 5 lines. This provides method chunks with enclosing class context.
func extractClassHeader(node *sitter.Node, content []byte) string {
	nodeContent := node.Content(content)
	lines := strings.SplitAfter(nodeContent, "\n")
	var headerLines []string
	for i, line := range lines {
		headerLines = append(headerLines, line)
		trimmed := strings.TrimRight(line, " \t\r\n")
		if strings.HasSuffix(trimmed, "{") ||
			strings.HasSuffix(trimmed, ":") ||
			strings.HasSuffix(trimmed, " do") ||
			trimmed == "do" {
			break
		}
		if i >= 4 {
			break
		}
	}
	return strings.Join(headerLines, "")
}
