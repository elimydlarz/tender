package tender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadTender_WithSpacesInName(t *testing.T) {
	root := t.TempDir()
	input := Tender{
		Name:   "Build Review",
		Agent:  "Build",
		Manual: true,
	}
	saved, err := SaveNewTender(root, input)
	if err != nil {
		t.Fatalf("SaveNewTender returned error: %v", err)
	}
	if saved.WorkflowFile == "" {
		t.Fatal("expected workflow filename")
	}

	path := filepath.Join(root, WorkflowDir, saved.WorkflowFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if got := string(data); !containsAll(got, `name: "tender/Build Review"`, `TENDER_NAME: "Build Review"`) {
		t.Fatalf("workflow missing expected name fields:\n%s", got)
	}

	list, err := LoadTenders(root)
	if err != nil {
		t.Fatalf("LoadTenders returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tender, got %d", len(list))
	}
	if list[0].Name != "Build Review" {
		t.Fatalf("unexpected name: %q", list[0].Name)
	}
}

func TestValidateTender_RejectsSlashInName(t *testing.T) {
	err := ValidateTender(Tender{
		Name:   "bad/name",
		Agent:  "Build",
		Manual: true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
