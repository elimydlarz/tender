package tender

import "sort"

const WorkflowDir = ".github/workflows"
const DefaultTimeoutMinutes = 30

type Tender struct {
	Name           string
	Agent          string
	Prompt         string
	Cron           string
	Manual         bool
	Push           bool
	TimeoutMinutes int
	WorkflowFile   string
}

func normalizeTimeoutMinutes(timeoutMinutes int) int {
	if timeoutMinutes > 0 {
		return timeoutMinutes
	}
	return DefaultTimeoutMinutes
}

func SortTenders(tenders []Tender) {
	sort.Slice(tenders, func(i, j int) bool {
		if tenders[i].Name == tenders[j].Name {
			return tenders[i].WorkflowFile < tenders[j].WorkflowFile
		}
		return tenders[i].Name < tenders[j].Name
	})
}
