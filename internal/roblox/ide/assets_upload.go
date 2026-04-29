package ide

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

var (
	ErrRateLimited = fmt.Errorf("rate limited")
	ErrBadRequest  = fmt.Errorf("bad request")
	ErrEmptyFile   = fmt.Errorf("file content is empty")
)

func setAPIKeyHeader(req *http.Request, apiKey string) {
	req.Header.Set("x-api-key", apiKey)
}

func newCreateAssetRequest(apiKey, assetType, name, description string, fileData []byte, fileName string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("assetType", assetType)
	_ = writer.WriteField("displayName", name)
	_ = writer.WriteField("description", description)

	part, err := writer.CreateFormFile("fileContent", fileName)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("write file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest("POST", "https://apis.roblox.com/assets/v1/assets", body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	setAPIKeyHeader(req, apiKey)

	return req, nil
}

func pollOperation(client *http.Client, apiKey, operationID string) (string, error) {
	url := fmt.Sprintf("https://apis.roblox.com/assets/v1/operations/%s", operationID)
	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", err
		}
		setAPIKeyHeader(req, apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			return "", ErrRateLimited
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("poll failed (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			Done     bool `json:"done"`
			Response struct {
				AssetID string `json:"assetId"`
			} `json:"response"`
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}

		if result.Done {
			if result.Error.Code != "" {
				if strings.Contains(result.Error.Message, "inappropriate") {
					return "", fmt.Errorf("inappropriate name or description")
				}
				return "", fmt.Errorf("operation failed: %s", result.Error.Message)
			}
			return result.Response.AssetID, nil
		}
		time.Sleep(2 * time.Second)
	}
}

func parseAssetID(resp *http.Response) (string, error) {
	var result struct {
		OperationID string `json:"operationId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode operation id: %w", err)
	}
	return result.OperationID, nil
}

func UploadAssetUsingOpenCloud(apiKey, assetType, name, description string, fileData []byte, fileName string) (string, error) {
	// Guard against truly empty files
	if len(fileData) == 0 {
		return "", fmt.Errorf("%w: asset file is empty (name=%q, type=%q)", ErrEmptyFile, name, assetType)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := newCreateAssetRequest(apiKey, assetType, name, description, fileData, fileName)
	if err != nil {
		return "", err
	}

	// ---------- DEBUG: log file and body sizes ----------
	bodySize := 0
	if req.Body != nil {
		// Read body to get length (will be rebuilt by http.NewRequest if needed, but we can just check the buffer)
		// Since we used a bytes.Buffer, we can cast it back to get the length.
		if buf, ok := req.Body.(*bytes.Buffer); ok {
			bodySize = buf.Len()
		}
	}
	fmt.Printf("DEBUG: uploading asset=%q type=%q fileSize=%d bodySize=%d\n", name, assetType, len(fileData), bodySize)
	// ----------------------------------------------------

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", ErrRateLimited
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "inappropriate") {
			return "", fmt.Errorf("inappropriate name or description")
		}
		return "", fmt.Errorf("create asset failed (%d): %s", resp.StatusCode, string(body))
	}

	operationID, err := parseAssetID(resp)
	if err != nil {
		return "", err
	}

	assetIDStr, err := pollOperation(client, apiKey, operationID)
	if err != nil {
		return "", err
	}
	return assetIDStr, nil
}
