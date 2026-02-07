package tender

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverPrimaryAgents_FromJSONConfig(t *testing.T) {
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")

	root := t.TempDir()
	cfg := `{
  // comment
  "default_agent": "default_primary",
  "agent": {
    "audit": {"mode": "primary"},
    "planner": {"mode": "all"},
    "helper": {"mode": "subagent"},
    "disabled_one": {"mode": "primary", "disable": true},
    "implicit_primary": {}
  }
}`
	if err := os.WriteFile(filepath.Join(root, "opencode.json"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := DiscoverPrimaryAgents(root)
	if err != nil {
		t.Fatalf("DiscoverPrimaryAgents: %v", err)
	}

	want := []string{"audit", "default_primary", "implicit_primary", "planner"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("agents mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestDiscoverPrimaryAgents_FromMarkdownAgents(t *testing.T) {
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")

	root := t.TempDir()
	agentDir := filepath.Join(root, ".opencode", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	files := map[string]string{
		"primary.md":        "---\nmode: primary\n---\n# primary\n",
		"all.md":            "---\nmode: all\n---\n# all\n",
		"sub.md":            "---\nmode: subagent\n---\n# sub\n",
		"disabled.md":       "---\nmode: primary\ndisabled: true\n---\n# disabled\n",
		"no_frontmatter.md": "# no frontmatter\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(agentDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	got, err := DiscoverPrimaryAgents(root)
	if err != nil {
		t.Fatalf("DiscoverPrimaryAgents: %v", err)
	}

	want := []string{"all", "no_frontmatter", "primary"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("agents mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestParseMarkdownAgentFrontmatter(t *testing.T) {
	t.Run("parses valid frontmatter with mode and disabled", func(t *testing.T) {
		content := `---
mode: primary
disabled: true
---
# Agent content`

		mode, disabled := parseMarkdownAgentFrontmatter(content)
		if mode != "primary" {
			t.Fatalf("expected mode 'primary', got %q", mode)
		}
		if !disabled {
			t.Fatal("expected disabled to be true")
		}
	})

	t.Run("handles different disabled values", func(t *testing.T) {
		cases := []struct {
			value    string
			expected bool
		}{
			{"true", true},
			{"yes", true},
			{"1", true},
			{"false", false},
			{"no", false},
			{"0", false},
			{"", false},
		}

		for _, tc := range cases {
			t.Run(tc.value, func(t *testing.T) {
				content := `---
disabled: ` + tc.value + `
---
# Agent content`

				_, disabled := parseMarkdownAgentFrontmatter(content)
				if disabled != tc.expected {
					t.Fatalf("expected disabled=%v for value %q, got %v", tc.expected, tc.value, disabled)
				}
			})
		}
	})

	t.Run("handles content without frontmatter", func(t *testing.T) {
		content := `# Just a markdown file
No frontmatter here.`

		mode, disabled := parseMarkdownAgentFrontmatter(content)
		if mode != "" {
			t.Fatalf("expected empty mode, got %q", mode)
		}
		if disabled {
			t.Fatal("expected disabled to be false")
		}
	})

	t.Run("handles malformed frontmatter", func(t *testing.T) {
		cases := []string{
			"---\ninvalid frontmatter\n---\n# Content",
			"---\nonlykey\n---\n# Content",
			"---\n: value\n---\n# Content",
			"---\nkey:\n---\n# Content", // missing value
		}

		for _, content := range cases {
			t.Run("malformed case", func(t *testing.T) {
				mode, disabled := parseMarkdownAgentFrontmatter(content)
				if mode != "" {
					t.Fatalf("expected empty mode for malformed frontmatter, got %q", mode)
				}
				if disabled {
					t.Fatal("expected disabled to be false for malformed frontmatter")
				}
			})
		}
	})

	t.Run("ignores comments and empty lines in frontmatter", func(t *testing.T) {
		content := `---
# This is a comment
mode: all

# Another comment
disabled: false
---
# Agent content`

		mode, disabled := parseMarkdownAgentFrontmatter(content)
		if mode != "all" {
			t.Fatalf("expected mode 'all', got %q", mode)
		}
		if disabled {
			t.Fatal("expected disabled to be false")
		}
	})

	t.Run("handles quoted values", func(t *testing.T) {
		content := `---
mode: "subagent"
disabled: "false"
---
# Agent content`

		mode, disabled := parseMarkdownAgentFrontmatter(content)
		if mode != "subagent" {
			t.Fatalf("expected mode 'subagent', got %q", mode)
		}
		if disabled {
			t.Fatal("expected disabled to be false for quoted 'false'")
		}
	})

	t.Run("handles BOM and whitespace", func(t *testing.T) {
		content := "\ufeff---\nmode: primary\n---\n# Content"

		mode, disabled := parseMarkdownAgentFrontmatter(content)
		if mode != "primary" {
			t.Fatalf("expected mode 'primary', got %q", mode)
		}
		if disabled {
			t.Fatal("expected disabled to be false")
		}
	})
}

func TestLoadAgentsFromJSONConfig(t *testing.T) {
	t.Run("handles empty config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "empty.json")
		content := `{}`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write empty config: %v", err)
		}

		set := map[string]bool{}
		err := loadAgentsFromJSONConfig(configPath, set)
		if err != nil {
			t.Fatalf("unexpected error for empty config: %v", err)
		}
		if len(set) != 0 {
			t.Fatalf("expected empty set, got %v", set)
		}
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		set := map[string]bool{}
		err := loadAgentsFromJSONConfig("/non/existent/file.json", set)
		if err != nil {
			t.Fatal("expected no error for non-existent file")
		}
	})

	t.Run("handles directory path", func(t *testing.T) {
		tmpDir := t.TempDir()
		set := map[string]bool{}
		err := loadAgentsFromJSONConfig(tmpDir, set)
		if err != nil {
			t.Fatal("expected no error for directory path")
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.json")
		invalidContent := `{"invalid": json}`
		if err := os.WriteFile(invalidPath, []byte(invalidContent), 0o644); err != nil {
			t.Fatalf("failed to write invalid JSON: %v", err)
		}

		set := map[string]bool{}
		err := loadAgentsFromJSONConfig(invalidPath, set)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed parsing") {
			t.Fatalf("expected parsing error, got: %v", err)
		}
	})
}

func TestLoadAgentsFromMarkdownDir(t *testing.T) {
	t.Run("handles non-existent directory", func(t *testing.T) {
		set := map[string]bool{}
		err := loadAgentsFromMarkdownDir("/non/existent/dir", set)
		if err != nil {
			t.Fatal("expected no error for non-existent directory")
		}
	})

	t.Run("ignores non-markdown files", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := []string{"agent.txt", "config.json", "script.sh"}
		for _, file := range files {
			path := filepath.Join(tmpDir, file)
			if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
				t.Fatalf("failed to write %s: %v", file, err)
			}
		}

		set := map[string]bool{}
		err := loadAgentsFromMarkdownDir(tmpDir, set)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(set) != 0 {
			t.Fatalf("expected empty set for non-markdown files, got %v", set)
		}
	})

	t.Run("handles unreadable files", func(t *testing.T) {
		tmpDir := t.TempDir()
		agentPath := filepath.Join(tmpDir, "unreadable.md")
		if err := os.WriteFile(agentPath, []byte("content"), 0o000); err != nil {
			t.Fatalf("failed to create unreadable file: %v", err)
		}
		defer os.Chmod(agentPath, 0o644)

		set := map[string]bool{}
		err := loadAgentsFromMarkdownDir(tmpDir, set)
		if err == nil {
			t.Fatal("expected error for unreadable file")
		}
	})
}

func TestStripJSONCComments(t *testing.T) {
	t.Run("removes line comments", func(t *testing.T) {
		in := `{"key": "value" // comment}`
		out := stripJSONCComments(in)
		expected := `{"key": "value" `
		if out != expected {
			t.Fatalf("expected %q, got %q", expected, out)
		}
	})

	t.Run("removes block comments", func(t *testing.T) {
		in := `{"key": /* comment */ "value"}`
		out := stripJSONCComments(in)
		expected := `{"key":  "value"}`
		if out != expected {
			t.Fatalf("expected %q, got %q", expected, out)
		}
	})

	t.Run("preserves string literals with comment patterns", func(t *testing.T) {
		in := `{"key": "value // not a comment", "other": "/* not a comment */"}`
		out := stripJSONCComments(in)
		if !strings.Contains(out, "value // not a comment") {
			t.Fatal("string content with // should be preserved")
		}
		if !strings.Contains(out, "/* not a comment */") {
			t.Fatal("string content with /* */ should be preserved")
		}
	})

	t.Run("handles escaped quotes in strings", func(t *testing.T) {
		in := `{"key": "value with \"escaped\" quotes" // comment}`
		out := stripJSONCComments(in)

		// The escaped quotes should be preserved in the output
		expected := `{"key": "value with \"escaped\" quotes" `
		if out != expected {
			t.Fatalf("expected %q, got %q", expected, out)
		}
		// Verify the comment was removed
		if strings.Contains(out, "// comment") {
			t.Fatal("comment should be removed")
		}
	})

	t.Run("handles complex nested comments", func(t *testing.T) {
		in := `{
  // line comment
  "key1": "value1",
  /* block comment
     spanning multiple lines */
  "key2": "value2"
}`
		out := stripJSONCComments(in)
		if strings.Contains(out, "// line comment") {
			t.Fatal("line comment should be removed")
		}
		if strings.Contains(out, "block comment") {
			t.Fatal("block comment should be removed")
		}
		if !strings.Contains(out, `"key1": "value1"`) {
			t.Fatal("valid JSON should be preserved")
		}
		if !strings.Contains(out, `"key2": "value2"`) {
			t.Fatal("valid JSON should be preserved")
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		out := stripJSONCComments("")
		if out != "" {
			t.Fatalf("expected empty string, got %q", out)
		}
	})

	t.Run("handles input without comments", func(t *testing.T) {
		in := `{"valid": "json", "number": 42}`
		out := stripJSONCComments(in)
		if out != in {
			t.Fatalf("input without comments should be unchanged: expected %q, got %q", in, out)
		}
	})
}
