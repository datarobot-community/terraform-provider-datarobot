package client

// ImportNotebookResponse represents the response from importing a notebook.
type ImportNotebookResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Notebook represents a notebook in DataRobot.
type Notebook struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UseCaseID string `json:"useCaseId"`
}
