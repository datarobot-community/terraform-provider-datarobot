package client

// TODO: Test what these actually are. The LLM hallucinated them
// ImportNotebookResponse represents the response from importing a notebook
type ImportNotebookResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Notebook represents a notebook in DataRobot.
type Notebook struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Created  string `json:"created"`
	Modified string `json:"modified"`
	Creator  string `json:"creator"`
}
