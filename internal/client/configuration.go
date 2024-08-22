package client

import "net/http"

const DefaultEndpoint string = "https://app.datarobot.com/api/v2"

// Configuration represents the configuration for the client.
type Configuration struct {
	UserAgent  string `json:"userAgent,omitempty"`
	Debug      bool   `json:"debug,omitempty"`
	Endpoint   string
	HTTPClient *http.Client
	token      string
}

// NewConfiguration returns a new Configuration object.
func NewConfiguration(token string) *Configuration {
	cfg := &Configuration{
		UserAgent: "datarobot-sdk-go/0.0.1",
		Debug:     false,
		Endpoint:  DefaultEndpoint,
		token:     token,
	}

	return cfg
}
