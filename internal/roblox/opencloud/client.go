package opencloud

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	OpenCloudAPIBase = "https://apis.roblox.com/assets/v1"
	UserAgent        = "AssetReuploader/2.0.0"
)

var (
	ErrNoAPIKey       = errors.New("no API key provided")
	ErrInvalidRequest = errors.New("invalid request")
	ErrUploadFailed   = errors.New("upload failed")
)

type UploadResponse struct {
	AssetID     int64  `json:"assetId"`
	AssetType   string `json:"assetType"`
	CreatedTime string `json:"createdTime"`
	VersionID   string `json:"versionId"`
}

type ErrorResponse struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details"`
}

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}

	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func (c *Client) UploadAsset(assetType string, data []byte, name string, description string) (*UploadResponse, error) {
	if len(data) == 0 {
		return nil, ErrInvalidRequest
	}

	url := fmt.Sprintf("%s/assets?request.display_name=%s", OpenCloudAPIBase, name)
	if description != "" {
		url += fmt.Sprintf("&request.description=%s", description)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", getMimeType(assetType))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("%s: %s (HTTP %d)", errResp.Code, errResp.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return nil, err
	}

	return &uploadResp, nil
}

func getMimeType(assetType string) string {
	switch assetType {
	case "Animation":
		return "model/x-rbxm"
	case "Mesh":
		return "model/x-file-mesh-data"
	case "Audio":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}
