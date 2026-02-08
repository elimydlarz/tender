package tender

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestDiscoverPrimaryAgents_FromCLIText(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
NAME MODE
Build primary
TestReviewer primary
Build primary
EOF
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := DiscoverPrimaryAgents(root)
	if err != nil {
		t.Fatalf("DiscoverPrimaryAgents: %v", err)
	}

	want := []string{"Build", "TestReviewer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("agents mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestDiscoverPrimaryAgents_FromCLIJSON(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
{"agents":[{"name":"Build"},{"name":"TestReviewer"}]}
EOF
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := DiscoverPrimaryAgents(root)
	if err != nil {
		t.Fatalf("DiscoverPrimaryAgents: %v", err)
	}

	want := []string{"Build", "TestReviewer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("agents mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestDiscoverPrimaryAgents_ErrorWhenMissingCLI(t *testing.T) {
	root := t.TempDir()
	emptyPath := t.TempDir()
	t.Setenv("PATH", emptyPath)

	_, err := DiscoverPrimaryAgents(root)
	if err == nil {
		t.Fatal("expected error when opencode is missing")
	}
	if !strings.Contains(err.Error(), "opencode agent list failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverPrimaryAgents_ErrorWhenNoAgents(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	writeFakeOpenCode(t, binDir, `#!/bin/sh
echo "NAME MODE"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := DiscoverPrimaryAgents(root)
	if err == nil {
		t.Fatal("expected error for empty agent list")
	}
	if !strings.Contains(err.Error(), "returned no usable agents") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOpenCodeAgentList_StripsANSIAndHeaders(t *testing.T) {
	out := "\x1b[38;5;45mNAME MODE\x1b[0m\nBuild primary\nTestReviewer primary\n"
	got := parseOpenCodeAgentList(out)
	want := []string{"Build", "TestReviewer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("agents mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func writeFakeOpenCode(t *testing.T, dir, script string) {
	t.Helper()

	name := "opencode"
	if runtime.GOOS == "windows" {
		name = "opencode.bat"
		script = "@echo off\r\n" + script + "\r\n"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
