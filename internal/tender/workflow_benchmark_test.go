package tender

import (
	"fmt"
	"testing"
)

// workflow_benchmark_test.go - Performance benchmarks for tender operations

func BenchmarkLoadTenders_Small(b *testing.B) {
	root := setupBenchmarkRepo(b, 10) // 10 tenders
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := LoadTenders(root)
		if err != nil {
			b.Fatalf("LoadTenders failed: %v", err)
		}
	}
}

func BenchmarkLoadTenders_Medium(b *testing.B) {
	root := setupBenchmarkRepo(b, 50) // 50 tenders
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := LoadTenders(root)
		if err != nil {
			b.Fatalf("LoadTenders failed: %v", err)
		}
	}
}

func BenchmarkLoadTenders_Large(b *testing.B) {
	root := setupBenchmarkRepo(b, 200) // 200 tenders
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := LoadTenders(root)
		if err != nil {
			b.Fatalf("LoadTenders failed: %v", err)
		}
	}
}

func BenchmarkSaveTender(b *testing.B) {
	root := b.TempDir()
	if err := EnsureWorkflowDir(root); err != nil {
		b.Fatalf("failed to create workflow dir: %v", err)
	}

	tender := Tender{
		Name:   "benchmark-test",
		Agent:  "Build",
		Manual: true,
		Cron:   "0 9 * * *",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Use unique name to avoid conflicts
		tender.Name = fmt.Sprintf("benchmark-test-%d", i)
		err := SaveTender(root, tender)
		if err != nil {
			b.Fatalf("SaveTender failed: %v", err)
		}
	}
}

func BenchmarkRenderWorkflow(b *testing.B) {
	tender := Tender{
		Name:   "benchmark-workflow",
		Agent:  "Build",
		Prompt: "This is a test prompt for benchmarking workflow rendering performance",
		Manual: true,
		Cron:   "30 14 * * 1,3,5",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = RenderWorkflow(tender)
	}
}

func BenchmarkParseTenderWorkflow(b *testing.B) {
	workflow := `name: "tender/benchmark-workflow"
on:
  workflow_dispatch:
    inputs:
      prompt:
        description: "Optional prompt override"
        required: false
        default: ""
        type: string
  schedule:
    - cron: "30 14 * * 1,3,5"
permissions:
  contents: write
concurrency:
  group: tender-main
  cancel-in-progress: false
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "benchmark-workflow"
      TENDER_AGENT: "Build"
      TENDER_PROMPT: "This is a test prompt for benchmarking"
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Install OpenCode
        shell: bash
        run: |
          set -euo pipefail
          curl -fsSL https://opencode.ai/install | bash
          echo "$HOME/bin" >> "$GITHUB_PATH"
          echo "$HOME/.local/bin" >> "$GITHUB_PATH"
          echo "$HOME/.opencode/bin" >> "$GITHUB_PATH"
      - name: Prepare main
        shell: bash
        run: |
          set -euo pipefail
          git config user.name "tender[bot]"
          git config user.email "tender[bot]@users.noreply.github.com"
          git fetch origin main
          git checkout -B main origin/main
      - name: Run OpenCode
        shell: bash
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          set -euo pipefail
          DISPATCH_PROMPT="${{ github.event_name == 'workflow_dispatch' && inputs.prompt || '' }}"
          RUN_PROMPT="${DISPATCH_PROMPT:-}"
          if [ -z "${RUN_PROMPT}" ]; then
            RUN_PROMPT="${TENDER_PROMPT:-}"
          fi
          if [ -z "${RUN_PROMPT}" ]; then
            RUN_PROMPT="Run the tender task '$TENDER_NAME' for this repository."
          fi
          opencode run --agent "$TENDER_AGENT" "$RUN_PROMPT"
      - name: Commit and push main
        shell: bash
        run: |
          set -euo pipefail
          CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD || echo detached)"
          AHEAD_COUNT="$(git rev-list --count origin/main..HEAD || echo 0)"
          if git diff --quiet --ignore-submodules -- && git diff --cached --quiet --ignore-submodules --; then
            if [ "$CURRENT_BRANCH" != "main" ] || [ "$AHEAD_COUNT" -gt 0 ]; then
              echo "No working tree changes; pushing existing commits from $CURRENT_BRANCH to main"
              git pull --rebase origin main
              git push origin HEAD:main
              exit 0
            fi
            echo "No changes to commit"
            exit 0
          fi
          git add -A
          git commit -m "tender($TENDER_NAME): autonomous update"
          git pull --rebase origin main
          git push origin HEAD:main`

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, ok := parseTenderWorkflow(workflow)
		if !ok {
			b.Fatal("failed to parse workflow")
		}
	}
}

func BenchmarkSortTenders(b *testing.B) {
	// Create unsorted slice of tenders
	tenders := make([]Tender, 100)
	for i := 0; i < 100; i++ {
		tenders[i] = Tender{
			Name:         fmt.Sprintf("tender-%03d", 99-i), // Reverse order
			Agent:        "Build",
			WorkflowFile: fmt.Sprintf("file-%03d.yml", 99-i),
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Copy to avoid modifying original
		testTenders := make([]Tender, len(tenders))
		copy(testTenders, tenders)
		SortTenders(testTenders)
	}
}

func BenchmarkFindTenderIndex(b *testing.B) {
	tenders := make([]Tender, 1000)
	for i := 0; i < 1000; i++ {
		tenders[i] = Tender{
			Name:  fmt.Sprintf("tender-%03d", i),
			Agent: "Build",
		}
	}

	// Test finding various tenders
	testCases := []string{"tender-000", "tender-500", "tender-999", "non-existent"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tc := testCases[i%len(testCases)]
		_ = findTenderIndex(tenders, tc)
	}
}

func BenchmarkTriggerSummary(b *testing.B) {
	testCases := []struct {
		cron   string
		manual bool
		push   bool
	}{
		{"", true, false},                        // Manual only
		{"0 9 * * *", false, false},              // Daily only
		{"30 14 * * 1,3,5", true, false},         // Weekly + manual
		{"*/15 * * * *", false, false},           // Hourly
		{"0 12 * * 0,1,2,3,4,5,6", false, false}, // Daily all days
		{"", true, true},                         // Manual + push
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tc := testCases[i%len(testCases)]
		_ = TriggerSummary(tc.cron, tc.manual, tc.push)
	}
}

func BenchmarkValidateTender(b *testing.B) {
	tender := Tender{
		Name:   "benchmark-validation",
		Agent:  "Build",
		Prompt: "Test prompt for validation benchmark",
		Manual: true,
		Cron:   "0 9 * * *",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := ValidateTender(tender)
		if err != nil {
			b.Fatalf("ValidateTender failed: %v", err)
		}
	}
}

func BenchmarkSlugify(b *testing.B) {
	testCases := []string{
		"Simple Name",
		"Name with spaces and special chars!@#$%",
		"MixedCASE_with-Different SEPARATORS",
		"123456789",
		"---leading---and---trailing---dashes---",
		"",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tc := testCases[i%len(testCases)]
		_ = Slugify(tc)
	}
}

func BenchmarkBuildCronExpressions(b *testing.B) {
	b.Run("Hourly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := buildHourlyCron("15")
			if err != nil {
				b.Fatalf("buildHourlyCron failed: %v", err)
			}
		}
	})

	b.Run("Daily", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := buildDailyCron("09:30")
			if err != nil {
				b.Fatalf("buildDailyCron failed: %v", err)
			}
		}
	})

	b.Run("Weekly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := buildWeeklyCron("1,3,5", "14:30")
			if err != nil {
				b.Fatalf("buildWeeklyCron failed: %v", err)
			}
		}
	})
}

// Helper function to set up benchmark repository with specified number of tenders
func setupBenchmarkRepo(b *testing.B, numTenders int) string {
	root := b.TempDir()
	if err := EnsureWorkflowDir(root); err != nil {
		b.Fatalf("failed to create workflow dir: %v", err)
	}

	for i := 0; i < numTenders; i++ {
		tender := Tender{
			Name:   fmt.Sprintf("tender-%03d", i),
			Agent:  []string{"Build", "Test", "Deploy"}[i%3],
			Manual: i%2 == 0,
			Cron:   fmt.Sprintf("%d %d * * *", i%60, i%24),
		}
		if _, err := SaveNewTender(root, tender); err != nil {
			b.Fatalf("failed to save benchmark tender %d: %v", i, err)
		}
	}

	return root
}

// Memory allocation benchmarks
func BenchmarkMemoryAllocation_RenderWorkflow(b *testing.B) {
	tender := Tender{
		Name:   "memory-test-workflow",
		Agent:  "Build",
		Prompt: "This is a test prompt to check memory allocation during workflow rendering",
		Manual: true,
		Cron:   "30 14 * * 1,3,5",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = RenderWorkflow(tender)
	}
}

func BenchmarkMemoryAllocation_LoadTenders(b *testing.B) {
	root := setupBenchmarkRepo(b, 50)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := LoadTenders(root)
		if err != nil {
			b.Fatalf("LoadTenders failed: %v", err)
		}
	}
}
