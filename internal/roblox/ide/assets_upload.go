package ide

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

// ErrRateLimited is returned when the Open Cloud API responds with 429.
var ErrRateLimited = errors.New("rate limited by Roblox Open Cloud API")

const openCloudAssetsURL = "https://apis.roblox.com/assets/v1/assets"

// openCloudOperationResponse is the async operation response from the Assets API.
type openCloudOperationResponse struct {
	Path     string `json:"path"`
	Done     bool   `json:"done"`
	Response *struct {
		AssetID int64 `json:"assetId,string"`
	} `json:"response,omitempty"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// UploadAnimationOpenCloud uploads an animation file via the Roblox Open Cloud
// Assets API using the provided API key.
// creatorType should be "User" or "Group", creatorID is the numeric ID.
// animData is the raw .rbxanim / KeyframeSequence bytes.
// Returns the new asset ID on success.
func UploadAnimationOpenCloud(
	client *http.Client,
	apiKey string,
	animName string,
	animData []byte,
	creatorType string,
	creatorID int64,
) (int64, error) {
	// Build multipart body
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	// Part 1: JSON request metadata
	metaHeader := textproto.MIMEHeader{}
	metaHeader.Set("Content-Type", "application/json")
	metaHeader.Set("Content-Disposition", `form-data; name="request"`)
	metaPart, err := mw.CreatePart(metaHeader)
	if err != nil {
		return 0, fmt.Errorf("creating metadata part: %w", err)
	}

	creatorField := "userId"
	if strings.EqualFold(creatorType, "Group") {
		creatorField = "groupId"
	}

	meta := map[string]interface{}{
		"assetType":   "Animation",
		"displayName": animName,
		"description": "",
		"creationContext": map[string]interface{}{
			"creator": map[string]interface{}{
				creatorField: creatorID,
			},
		},
	}
	if err := json.NewEncoder(metaPart).Encode(meta); err != nil {
		return 0, fmt.Errorf("encoding metadata: %w", err)
	}

	// Part 2: binary animation data
	fileHeader := textproto.MIMEHeader{}
	fileHeader.Set("Content-Type", "model/x-rbxm")
	fileHeader.Set("Content-Disposition", `form-data; name="fileContent"; filename="animation.rbxm"`)
	filePart, err := mw.CreatePart(fileHeader)
	if err != nil {
		return 0, fmt.Errorf("creating file part: %w", err)
	}
	if _, err := io.Copy(filePart, bytes.NewReader(animData)); err != nil {
		return 0, fmt.Errorf("writing animation data: %w", err)
	}

	if err := mw.Close(); err != nil {
		return 0, fmt.Errorf("closing multipart writer: %w", err)
	}

	// Send upload request
	req, err := http.NewRequest(http.MethodPost, openCloudAssetsURL, &body)
	if err != nil {
		return 0, fmt.Errorf("building upload request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return 0, ErrRateLimited
	case http.StatusForbidden, http.StatusUnauthorized:
		return 0, fmt.Errorf("API key rejected (HTTP %d) — make sure it has Assets Write permission", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	// Decode operation response
	rawUpload, _ := io.ReadAll(resp.Body)
	var op openCloudOperationResponse
	if err := json.Unmarshal(rawUpload, &op); err != nil {
		return 0, fmt.Errorf("decoding operation response: %w", err)
	}

	if op.Done {
		return extractAssetID(op)
	}

	if op.Path == "" {
		return 0, errors.New("empty operation path in response")
	}
	return pollOperation(client, apiKey, op.Path)
}

// pollOperation polls the async operation endpoint until Done == true.
func pollOperation(client *http.Client, apiKey, opPath string) (int64, error) {
	pollURL := "https://apis.roblox.com/assets/v1/" + opPath
	const maxAttempts = 20
	backoff := 500 * time.Millisecond

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(backoff)
		if backoff < 5*time.Second {
			backoff *= 2
		}

		req, err := http.NewRequest(http.MethodGet, pollURL, http.NoBody)
		if err != nil {
			return 0, err
		}
		req.Header.Set("x-api-key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		rawBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var op openCloudOperationResponse
		if err := json.Unmarshal(rawBody, &op); err != nil {
			continue
		}

		if op.Done {
			return extractAssetID(op)
		}
	}
	return 0, errors.New("timed out waiting for upload operation to complete")
}

func extractAssetID(op openCloudOperationResponse) (int64, error) {
	if op.Error != nil {
		return 0, fmt.Errorf("upload rejected by Roblox: %s", op.Error.Message)
	}
	if op.Response == nil || op.Response.AssetID == 0 {
		return 0, errors.New("operation succeeded but returned no asset ID")
	}
	return op.Response.AssetID, nil
}