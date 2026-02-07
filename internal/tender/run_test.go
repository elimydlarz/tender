package tender

import (
	"reflect"
	"testing"
)

func TestBuildGHWorkflowRunArgs_NoPrompt(t *testing.T) {
	got := buildGHWorkflowRunArgs(Tender{WorkflowFile: "nightly.yml"}, "")
	want := []string{"workflow", "run", "nightly.yml"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got=%v want=%v", got, want)
	}
}

func TestBuildGHWorkflowRunArgs_WithPrompt(t *testing.T) {
	got := buildGHWorkflowRunArgs(Tender{WorkflowFile: "nightly.yml"}, "Fix tests")
	want := []string{"workflow", "run", "nightly.yml", "-f", "prompt=Fix tests"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got=%v want=%v", got, want)
	}
}
