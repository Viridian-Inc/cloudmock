package cicd

// Store defines the interface for CI/CD pipeline and test result persistence.
type Store interface {
	SavePipeline(p Pipeline) error
	GetPipeline(id string) (*Pipeline, error)
	ListPipelines(limit int) []Pipeline
	SaveTestResults(pipelineID string, results []TestResult) error
	GetTestResults(pipelineID string) ([]TestResult, error)
	Summary() CISummary
}
