package client

type CreatePlaygroundRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UseCaseID   string `json:"useCaseId"`
}

type UpdatePlaygroundRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreatePlaygroundResponse struct {
	ID string `json:"id"`
}

type PlaygroundResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UseCaseID   string `json:"useCaseId"`
}
