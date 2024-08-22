package client

type CreateChatApplicationRequest struct {
	DeploymentID string `json:"deploymentId"`
}

type ChatApplicationResponse struct {
	ID                               string `json:"id"`
	Name                             string `json:"name"`
	Status                           string `json:"status"`
	CustomApplicationSourceID        string `json:"customApplicationSourceId"`
	CustomApplicationSourceVersionID string `json:"customApplicationSourceVersionId"`
	ApplicationUrl                   string `json:"applicationUrl"`
}

type UpdateChatApplicationRequest struct {
	Name string `json:"name"`
}

type ChatApplicationSourceResponse struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name"`
	LatestVersion ChatApplicationSourceVersion `json:"latestVersion"`
}

type ChatApplicationSourceVersion struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
