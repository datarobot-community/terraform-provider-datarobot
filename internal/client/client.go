package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type Client struct {
	cfg *Configuration
}

func NewClient(cfg *Configuration) *Client {
	if cfg == nil {
		panic("configuration is required")
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	return &Client{cfg: cfg}
}

func (c *Client) addAuthHeader(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+c.cfg.token)
}

func doRequest[T any](c *Client, ctx context.Context, method, apiPath string, body any) (result *T, err error) {
	// Marshal the body to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return result, WrapGenericError("failed to marshal body", err)
	}

	// Create a new HTTP request
	url := c.cfg.Endpoint + apiPath
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
	req = req.WithContext(ctx)
	if err != nil {
		return result, WrapGenericError("failed to create request", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	c.addAuthHeader(req)

	// Perform the request
	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		errorMessage := fmt.Sprintf("%s request %s failed", req.Method, req.URL.String())
		if c.cfg.Debug {
			errorMessage += getCurlCommand(req, jsonBody)
		}
		return result, WrapGenericError(errorMessage, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return result, NewNotFoundError(req.URL.String())
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return result, NewUnauthorizedError(req.URL.String())
	}

	// Read the response body
	var respBody []byte
	if resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return result, WrapGenericError("failed to read response body", err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorMessage := fmt.Sprintf("%s request %s : response %s %s", req.Method, req.URL.String(), resp.Status, string(respBody))
		if c.cfg.Debug {
			errorMessage += getCurlCommand(req, jsonBody)
		}
		return result, NewGenericError(errorMessage)
	}

	if (req.Method == http.MethodDelete || req.Method == http.MethodPatch) &&
		resp.StatusCode == http.StatusNoContent {
		return result, nil
	}

	if req.Method == http.MethodPatch && resp.StatusCode == http.StatusAccepted {
		return result, nil
	}

	if c.cfg.Debug {
		fmt.Printf("Request %s %s - Response %s %s\n\n", req.Method, req.URL.String(), resp.Status, string(respBody))
	}

	// Deserialize the response into the provided result type
	result = new(T)
	if err := json.Unmarshal(respBody, result); err != nil {
		return result, WrapGenericError("failed to unmarshal response", err)
	}

	return result, nil
}

func Get[T any](c *Client, ctx context.Context, path string) (*T, error) {
	return doRequest[T](c, ctx, http.MethodGet, path, nil)
}

func Post[T any](c *Client, ctx context.Context, path string, body any) (*T, error) {
	return doRequest[T](c, ctx, http.MethodPost, path, body)
}

func Put[T any](c *Client, ctx context.Context, path string, body any) (*T, error) {
	return doRequest[T](c, ctx, http.MethodPut, path, body)
}

func Delete(c *Client, ctx context.Context, path string) (err error) {
	_, err = doRequest[CreateVoidResponse](c, ctx, http.MethodDelete, path, nil)
	return
}

func Patch[T any](c *Client, ctx context.Context, path string, body any) (*T, error) {
	return doRequest[T](c, ctx, http.MethodPatch, path, body)
}

func getCurlCommand(req *http.Request, jsonBody []byte) string {
	curlCommand := fmt.Sprintf("curl -X %s '%s' ", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			curlCommand += fmt.Sprintf(" -H '%s: %s'", key, value)
		}
	}
	if jsonBody != nil {
		curlCommand += fmt.Sprintf(" -d '%s'", jsonBody)
	}
	return fmt.Sprintf(" (%s)", curlCommand)
}

type FileInfo struct {
	Name    string `json:"name,omitempty"`
	Path    string `json:"path,omitempty"`
	Content []byte `json:"content,omitempty"`
}

func uploadFileFromBinary[T any](
	c *Client,
	ctx context.Context,
	apiPath string,
	httpMethod string,
	fileName string,
	fileData []byte,
	otherFields map[string]string,
) (
	result *T,
	err error,
) {
	return uploadFilesFromBinaries[T](
		c,
		ctx,
		apiPath,
		httpMethod,
		[]FileInfo{{Name: fileName, Content: fileData}},
		otherFields,
	)
}

func uploadFilesFromBinaries[T any](
	c *Client,
	ctx context.Context,
	apiPath string,
	httpMethod string,
	files []FileInfo,
	otherFields map[string]string,
) (
	result *T,
	err error,
) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add other fields to the form
	for key, value := range otherFields {
		err := writer.WriteField(key, value)
		if err != nil {
			return result, WrapGenericError("could not write field", err)
		}
	}

	for _, file := range files {
		// Create the form file field
		part, err := writer.CreateFormFile("file", file.Name)
		if err != nil {
			return result, WrapGenericError("could not create form file", err)
		}

		// Write the file data to the form field
		_, err = part.Write(file.Content)
		if err != nil {
			return result, WrapGenericError("could not write file data", err)
		}

		if file.Path != "" {
			// Write the file path to the form field
			err = writer.WriteField("filePath", file.Path)
			if err != nil {
				return result, WrapGenericError("could not write file path", err)
			}
		}
	}

	// Close the multipart writer to set the terminating boundary
	err = writer.Close()
	if err != nil {
		return result, WrapGenericError("could not close writer", err)
	}

	// Create a new HTTP request
	url := c.cfg.Endpoint + apiPath
	req, err := http.NewRequestWithContext(ctx, httpMethod, url, &requestBody)
	req = req.WithContext(ctx)
	if err != nil {
		return result, WrapGenericError("failed to create request", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.addAuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return result, WrapGenericError("could not send request", err)
	}
	defer resp.Body.Close()

	var respBody []byte
	if resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return result, WrapGenericError("failed to read response body", err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorMessage := fmt.Sprintf("%s request %s : response %s %s", req.Method, req.URL.String(), resp.Status, string(respBody))
		return result, NewGenericError(errorMessage)
	}
	if c.cfg.Debug {
		fmt.Printf("Request %s %s - Response %s %s\n\n", req.Method, req.URL.String(), resp.Status, string(respBody))
	}

	// Deserialize the response into the provided result type
	result = new(T)
	if err := json.Unmarshal(respBody, result); err != nil {
		return result, WrapGenericError("failed to unmarshal response", err)
	}

	return result, nil
}
