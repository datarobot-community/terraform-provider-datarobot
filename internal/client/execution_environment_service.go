package client

type CreateExecutionEnvironmentRequest struct {
	Name                string   `json:"name"`
	Description         string   `json:"description,omitempty"`
	ProgrammingLanguage string   `json:"programmingLanguage"`
	UseCases            []string `json:"useCases"`
}

type CreateExecutionEnvironmentVersionRequest struct {
	Description    string     `json:"description,omitempty"`
	DockerImageUri string     `json:"dockerImageUri,omitempty"`
	Files          []FileInfo `json:"files"`
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
	IsPublic            bool                        `json:"isPublic"`
}

type ExecutionEnvironmentVersion struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	EnvironmentID  string `json:"environmentId"`
	ImageID        string `json:"imageId"`
	Description    string `json:"description"`
	BuildStatus    string `json:"buildStatus"`
	DockerImageUri string `json:"sourceDockerImageUri"`
	// BuildID identifies the OTel-logged build for this version. It becomes available
	// some time after the build starts, so it may still be empty (e.g. a very fast
	// failure) even once BuildStatus has moved past "processing".
	BuildID string `json:"buildId,omitempty"`
}

// ExecutionEnvironmentBuildLog is the response shape of the legacy per-version build
// log endpoint, kept as a fallback behind the newer OTel logs pipeline.
type ExecutionEnvironmentBuildLog struct {
	Log   string `json:"log"`
	Error string `json:"error,omitempty"`
}
