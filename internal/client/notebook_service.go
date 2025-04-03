package client

import "fmt"

// ImportNotebookResponse represents the response from importing a notebook.
type ImportNotebookResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"-"`
}

// Notebook represents a notebook in DataRobot.
type Notebook struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UseCaseID string `json:"useCaseId"`
	URL       string `json:"-"`
}

func URLForNotebook(notebookID string, useCaseID string, baseUrl string) string {
	// Sample with use case: usecases/67b9369bb3ec5426f0e97192/notebooks/67eb6c379ba5bc616450b482
	// Sample without use case: notebooks/67eb6c379ba5bc616450b482
	var useCasePath = ""
	if useCaseID != "" {
		useCasePath = fmt.Sprintf("/usecases/%s/", useCaseID)
	}

	return fmt.Sprintf("%s%s/notebooks/%s", baseUrl, useCasePath, notebookID)
}
