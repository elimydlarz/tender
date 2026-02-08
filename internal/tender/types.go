package tender

import "sort"

const WorkflowDir = ".github/workflows"

type Tender struct {
	Name         string
	Agent        string
	Prompt       string
	Cron         string
	Manual       bool
	Push         bool
	WorkflowFile string
}

func SortTenders(tenders []Tender) {
	sort.Slice(tenders, func(i, j int) bool {
		if tenders[i].Name == tenders[j].Name {
			return tenders[i].WorkflowFile < tenders[j].WorkflowFile
		}
		return tenders[i].Name < tenders[j].Name
	})
}
