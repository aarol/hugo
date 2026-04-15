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
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func TestHighlight(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		lang     string
		wantErr  bool
		checkFor []string // strings that should appear in output
	}{
		{
			name: "rust function",
			code: `fn main() { println!("Hello world"); }`,
			lang: "rust",
			checkFor: []string{
				`class="ts-keyword"`,
				`class="ts-function"`,
				"fn",
				"main",
			},
		},
		{
			name: "go function",
			code: `package main

func main() {
	fmt.Println("Hello, World!")
}`,
			lang: "go",
			checkFor: []string{
				`class="ts-keyword"`,
				"package",
				"func",
				"main",
			},
		},
		{
			name: "golang alias",
			code: `package main`,
			lang: "golang",
			checkFor: []string{
				`class="ts-keyword"`,
				"package",
			},
		},
		{
			name: "python function",
			code: `def hello():
    print("Hello, World!")`,
			lang: "python",
			checkFor: []string{
				`class="ts-keyword"`,
				"def",
				"hello",
			},
		},
		{
			name: "javascript function",
			code: `function hello() {
    console.log("Hello, World!");
}`,
			lang: "javascript",
			checkFor: []string{
				`class="ts-keyword"`,
				"function",
				"hello",
			},
		},
		{
			name: "typescript",
			code: `const greet = (name: string): void => {
    console.log(name);
}`,
			lang: "typescript",
			checkFor: []string{
				`ts-type`,
				"const",
				"greet",
			},
		},
		{
			name: "json",
			code: `{
    "name": "test",
    "value": 123
}`,
			lang: "json",
			checkFor: []string{
				`class="ts-string"`,
				`class="ts-number"`,
				"name",
				"test",
				"123",
			},
		},
		{
			name:     "empty code",
			code:     "",
			lang:     "go",
			checkFor: []string{},
		},
		{
			name: "unknown language",
			code: `some code here`,
			lang: "unknownlang123",
			checkFor: []string{
				"some code here",
			},
		},
		{
			name: "html escaping",
			code: `<script>alert("xss")</script>`,
			lang: "javascript",
			checkFor: []string{
				"&lt;",
				"&gt;",
				"alert",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := Highlight(tt.code, tt.lang)
			if (err != nil) != tt.wantErr {
				t.Errorf("Highlight() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.code == "" && html != "" {
				t.Errorf("Expected empty output for empty code, got: %s", html)
				return
			}

			for _, check := range tt.checkFor {
				if !strings.Contains(html, check) {
					t.Errorf("Expected output to contain %q, got: %s", check, html)
				}
			}

			// Log output for inspection
			if testing.Verbose() {
				t.Logf("Language: %s", tt.lang)
				t.Logf("Output HTML:\n%s", html)
			}
		})
	}
}

func TestCaptureToClass(t *testing.T) {
	tests := []struct {
		capture string
		want    string
	}{
		{"keyword", "ts-keyword"},
		{"keyword.function", "ts-keyword ts-function"},
		{"variable.builtin", "ts-variable ts-builtin"},
		{"string.special", "ts-string ts-special"},
		{"comment.line", "ts-comment ts-line"},
		{"constant_builtin", "ts-constant-builtin"},
	}

	for _, tt := range tests {
		t.Run(tt.capture, func(t *testing.T) {
			got := captureToClass(tt.capture)
			if got != tt.want {
				t.Errorf("captureToClass(%q) = %q, want %q", tt.capture, got, tt.want)
			}
		})
	}
}

func TestRangesToHTML(t *testing.T) {
	src := []byte("hello world")
	ranges := []gotreesitter.HighlightRange{
		{
			Capture:   "keyword",
			StartByte: 0,
			EndByte:   5,
		},
		{
			Capture:   "string",
			StartByte: 6,
			EndByte:   11,
		},
	}

	html := rangesToHTML(src, ranges)

	expected := `<span class="ts-keyword">hello</span> <span class="ts-string">world</span>`
	if html != expected {
		t.Errorf("rangesToHTML() = %q, want %q", html, expected)
	}
}

func TestRangesToHTMLWithGaps(t *testing.T) {
	src := []byte("hello beautiful world")
	ranges := []gotreesitter.HighlightRange{
		{
			Capture:   "keyword",
			StartByte: 0,
			EndByte:   5,
		},
		{
			Capture:   "string",
			StartByte: 16,
			EndByte:   21,
		},
	}

	html := rangesToHTML(src, ranges)

	// Should have "hello" highlighted, " beautiful " unhighlighted, and "world" highlighted
	if !strings.Contains(html, `<span class="ts-keyword">hello</span>`) {
		t.Errorf("Expected highlighted 'hello', got: %s", html)
	}
	if !strings.Contains(html, ` beautiful `) {
		t.Errorf("Expected unhighlighted ' beautiful ', got: %s", html)
	}
	if !strings.Contains(html, `<span class="ts-string">world</span>`) {
		t.Errorf("Expected highlighted 'world', got: %s", html)
	}
}

func TestRangesToHTMLWithHTMLEscaping(t *testing.T) {
	src := []byte(`<tag attr="value">`)
	ranges := []gotreesitter.HighlightRange{
		{
			Capture:   "tag",
			StartByte: 0,
			EndByte:   4,
		},
	}

	html := rangesToHTML(src, ranges)

	// HTML entities should be escaped
	if !strings.Contains(html, "&lt;tag") {
		t.Errorf("Expected HTML-escaped output, got: %s", html)
	}
	if strings.Contains(html, "<tag") {
		t.Errorf("Output should not contain raw HTML tags, got: %s", html)
	}
}
