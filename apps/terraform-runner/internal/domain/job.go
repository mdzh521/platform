package domain

type JobClaim struct {
	JobID         uint
	Name          string
	Provider      string
	Action        string
	AccountID     uint
	BlueprintID   uint
	NetworkPlanID *uint
	BlueprintCode string
	TemplatePath  string
	Input         map[string]any
	Environment   map[string]string
	WorkspacePath string
}

type JobResult struct {
	JobID        uint
	Status       string
	LogExcerpt   string
	Output       map[string]any
	PlanSummary  map[string]any
	ErrorMessage string
}
