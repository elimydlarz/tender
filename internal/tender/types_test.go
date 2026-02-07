package tender

import (
	"testing"
)

// types.go tests

func TestSortTenders(t *testing.T) {
	t.Run("sorts by name then workflow file", func(t *testing.T) {
		tenders := []Tender{
			{Name: "zebra", Agent: "Build", WorkflowFile: "a.yml"},
			{Name: "apple", Agent: "Test", WorkflowFile: "z.yml"},
			{Name: "banana", Agent: "Deploy", WorkflowFile: "m.yml"},
			{Name: "apple", Agent: "Build", WorkflowFile: "a.yml"},
		}

		SortTenders(tenders)

		expected := []Tender{
			{Name: "apple", Agent: "Build", WorkflowFile: "a.yml"},
			{Name: "apple", Agent: "Test", WorkflowFile: "z.yml"},
			{Name: "banana", Agent: "Deploy", WorkflowFile: "m.yml"},
			{Name: "zebra", Agent: "Build", WorkflowFile: "a.yml"},
		}

		if len(tenders) != len(expected) {
			t.Fatalf("expected %d tenders, got %d", len(expected), len(tenders))
		}

		for i, tender := range tenders {
			if tender.Name != expected[i].Name {
				t.Fatalf("tender %d: expected name %q, got %q", i, expected[i].Name, tender.Name)
			}
			if tender.WorkflowFile != expected[i].WorkflowFile {
				t.Fatalf("tender %d: expected workflow file %q, got %q", i, expected[i].WorkflowFile, tender.WorkflowFile)
			}
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		tenders := []Tender{}
		SortTenders(tenders)

		if len(tenders) != 0 {
			t.Fatalf("expected empty list, got %d tenders", len(tenders))
		}
	})

	t.Run("handles single item", func(t *testing.T) {
		tenders := []Tender{
			{Name: "single", Agent: "Build", WorkflowFile: "test.yml"},
		}
		SortTenders(tenders)

		if len(tenders) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(tenders))
		}
		if tenders[0].Name != "single" {
			t.Fatalf("expected name 'single', got %q", tenders[0].Name)
		}
	})

	t.Run("handles already sorted list", func(t *testing.T) {
		tenders := []Tender{
			{Name: "apple", Agent: "Build", WorkflowFile: "a.yml"},
			{Name: "banana", Agent: "Test", WorkflowFile: "b.yml"},
			{Name: "cherry", Agent: "Deploy", WorkflowFile: "c.yml"},
		}

		SortTenders(tenders)

		// Should remain the same
		expected := []string{"apple", "banana", "cherry"}
		for i, tender := range tenders {
			if tender.Name != expected[i] {
				t.Fatalf("tender %d: expected name %q, got %q", i, expected[i], tender.Name)
			}
		}
	})

	t.Run("sorts case-sensitively by name", func(t *testing.T) {
		tenders := []Tender{
			{Name: "Zebra", Agent: "Build", WorkflowFile: "a.yml"},
			{Name: "apple", Agent: "Test", WorkflowFile: "z.yml"},
			{Name: "Banana", Agent: "Deploy", WorkflowFile: "m.yml"},
		}

		SortTenders(tenders)

		// Go string comparison is case-sensitive, uppercase comes before lowercase
		expected := []string{"Banana", "Zebra", "apple"}
		for i, tender := range tenders {
			if tender.Name != expected[i] {
				t.Fatalf("tender %d: expected name %q, got %q", i, expected[i], tender.Name)
			}
		}
	})

	t.Run("uses workflow file as tiebreaker for same names", func(t *testing.T) {
		tenders := []Tender{
			{Name: "duplicate", Agent: "Build", WorkflowFile: "z.yml"},
			{Name: "duplicate", Agent: "Test", WorkflowFile: "a.yml"},
			{Name: "duplicate", Agent: "Deploy", WorkflowFile: "m.yml"},
		}

		SortTenders(tenders)

		expected := []string{"a.yml", "m.yml", "z.yml"}
		for i, tender := range tenders {
			if tender.WorkflowFile != expected[i] {
				t.Fatalf("tender %d: expected workflow file %q, got %q", i, expected[i], tender.WorkflowFile)
			}
		}
	})
}
