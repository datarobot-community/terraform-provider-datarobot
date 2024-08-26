package client

type CreateUseCaseResponse struct {
	ID string `json:"id"`
}

type UseCaseResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UseCaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
