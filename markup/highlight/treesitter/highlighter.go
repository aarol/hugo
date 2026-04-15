// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package treesitter

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// languageAliases maps common language name variations to canonical names
var languageAliases = map[string]string{
	"golang":      "go",
	"js":          "javascript",
	"ts":          "typescript",
	"py":          "python",
	"rb":          "ruby",
	"sh":          "bash",
	"shell":       "bash",
	"yml":         "yaml",
	"c++":         "cpp",
	"c#":          "c_sharp",
	"csharp":      "c_sharp",
	"objective-c": "objc",
	"objc":        "objc",
}

// getLanguageByName looks up a language entry by name from all registered languages.
func getLanguageByName(name string) *grammars.LangEntry {
	allLangs := grammars.AllLanguages()
	for i := range allLangs {
		if allLangs[i].Name == name {
			return &allLangs[i]
		}
	}
	return nil
}

// Highlight performs syntax highlighting on the given code using gotreesitter.
// Returns HTML with span elements containing CSS classes based on tree-sitter captures.
func Highlight(code, lang string) (string, error) {
	if code == "" {
		return "", nil
	}

	lang = strings.ToLower(lang)
	if alias, ok := languageAliases[lang]; ok {
		lang = alias
	}

	entry := getLanguageByName(lang)
	if entry == nil {
		entry = grammars.DetectLanguage("file." + lang)
	}
	if entry == nil || entry.Quality == "none" {
		return html.EscapeString(code), nil
	}

	language := entry.Language()
	if language == nil || entry.HighlightQuery == "" {
		return html.EscapeString(code), nil
	}

	// Create a new parser per call: parsers are not goroutine-safe.
	parser := gotreesitter.NewParser(language)
	srcBytes := []byte(code)
	tree, err := parser.Parse(srcBytes)
	if err != nil || tree == nil {
		return html.EscapeString(code), nil
	}

	highlighter, err := gotreesitter.NewHighlighter(language, entry.HighlightQuery)
	if err != nil {
		return html.EscapeString(code), nil
	}

	return rangesToHTML(srcBytes, highlighter.Highlight(srcBytes)), nil
}

// rangesToHTML converts highlight ranges to HTML with span elements.
// Each range gets a span with a class based on the capture name.
func rangesToHTML(src []byte, ranges []gotreesitter.HighlightRange) string {
	if len(ranges) == 0 {
		return html.EscapeString(string(src))
	}

	var buf bytes.Buffer
	lastPos := uint32(0)

	for _, r := range ranges {
		// Write any unhighlighted text before this range
		if r.StartByte > lastPos {
			text := src[lastPos:r.StartByte]
			buf.WriteString(html.EscapeString(string(text)))
		}

		// Write the highlighted span
		text := src[r.StartByte:r.EndByte]
		className := captureToClass(r.Capture)

		// Only add span if we have a class (skip empty classes)
		if className != "" {
			fmt.Fprintf(&buf, `<span class="%s">%s</span>`,
				className,
				html.EscapeString(string(text)))
		} else {
			buf.WriteString(html.EscapeString(string(text)))
		}

		lastPos = r.EndByte
	}

	// Write any remaining unhighlighted text
	if int(lastPos) < len(src) {
		text := src[lastPos:]
		buf.WriteString(html.EscapeString(string(text)))
	}

	return buf.String()
}

// captureToClass converts a tree-sitter capture name to a CSS class name with ts- prefix.
func captureToClass(capture string) string {
	// Handle scoped captures (e.g., "keyword.function" -> "ts-keyword ts-function")
	parts := strings.Split(capture, ".")

	// Build class name from parts
	var classes []string
	for _, part := range parts {
		// Normalize part (replace underscores with hyphens for CSS)
		normalized := strings.ReplaceAll(part, "_", "-")
		classes = append(classes, normalized)
	}

	// Return space-separated classes with "ts-" prefix
	classStr := strings.Join(classes, " ts-")
	return "ts-" + classStr
}
