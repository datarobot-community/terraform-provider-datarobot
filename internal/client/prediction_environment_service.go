package client

type CreatePredictionEnvironmentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Platform    string `json:"platform"`
}

type UpdatePredictionEnvironmentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PredictionEnvironment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Platform    string `json:"platform"`
}
