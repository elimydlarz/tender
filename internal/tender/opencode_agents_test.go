package tender

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestStripJSONCComments(t *testing.T) {
	in := `{
  "url": "https://example.com/x", // keep URL
  /* drop block */
  "agent": {"a": {"mode": "primary"}}
}`
	out := stripJSONCComments(in)
	if out == in {
		t.Fatalf("expected comments to be stripped")
	}
	if reflect.DeepEqual(out, "") {
		t.Fatalf("stripped output should not be empty")
	}
}
