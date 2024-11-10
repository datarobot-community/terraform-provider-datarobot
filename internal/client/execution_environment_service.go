package client

type CreateExecutionEnvironmentRequest struct {
	Name                string   `json:"name"`
	Description         string   `json:"description,omitempty"`
	ProgrammingLanguage string   `json:"programmingLanguage"`
	UseCases            []string `json:"useCases"`
}

type CreateExecutionEnvironmentVersionRequest struct {
	Description string     `json:"description,omitempty"`
	Files       []FileInfo `json:"files"`
}

type UpdateExecutionEnvironmentRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	UseCases    []string `json:"useCases"`
}

type ExecutionEnvironment struct {
	ID                  string                      `json:"id"`
	Name                string                      `json:"name"`
	Description         string                      `json:"description"`
	ProgrammingLanguage string                      `json:"programmingLanguage"`
	UseCases            []string                    `json:"useCases"`
	LatestVersion       ExecutionEnvironmentVersion `json:"latestVersion"`
}

type ExecutionEnvironmentVersion struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	EnvironmentID string `json:"environmentId"`
	ImageID       string `json:"imageId"`
	Description   string `json:"description"`
	BuildStatus   string `json:"buildStatus"`
}
