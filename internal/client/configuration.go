package client

import (
	"net/http"
	"net/url"
)

const DefaultEndpoint string = "https://app.datarobot.com/api/v2"

// Configuration represents the configuration for the client.
type Configuration struct {
	UserAgent    string `json:"userAgent,omitempty"`
	Debug        bool   `json:"debug,omitempty"`
	Endpoint     string
	TraceContext string
	HTTPClient   *http.Client
	token        string
}

// BaseURL returns the base URL (without the API path) derived from the Endpoint.
func (c *Configuration) BaseURL() string {
	parsedURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return ""
	}
	u := &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
	}
	return u.String()
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
