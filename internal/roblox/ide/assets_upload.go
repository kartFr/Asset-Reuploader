package ide

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/kartFr/Asset-Reuploader/internal/app/config"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

const (
	createAssetURL   = "https://apis.roblox.com/assets/v1/assets"
	operationBaseURL = "https://apis.roblox.com/assets/v1/operations/"
	maxPollAttempts  = 30
	pollInterval     = time.Second
)

var errTokenInvalid = errors.New("XSRF token validation failed")
var ErrRateLimited = errors.New("rate limited")

func setAPIKeyHeader(req *http.Request) {
	apiKey := strings.TrimSpace(config.Get("api_key"))
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
}

type createAssetRequest struct {
	AssetType       string                `json:"assetType"`
	DisplayName     string                `json:"displayName"`
	Description     string                `json:"description"`
	CreationContext createCreationContext `json:"creationContext"`
}

type createCreationContext struct {
	Creator createCreator `json:"creator"`
}

type createCreator struct {
	UserID  int64 `json:"userId,omitempty"`
	GroupID int64 `json:"groupId,omitempty"`
}

type operationResponse struct {
	Path     string            `json:"path"`
	Done     bool              `json:"done"`
	Error    *operationError   `json:"error,omitempty"`
	Response *operationAssetID `json:"response,omitempty"`
}

type operationError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type operationAssetID struct {
	AssetID string `json:"assetId"`
	Path    string `json:"path"`
}

type statusResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newCreateAssetRequest(
	assetType string,
	name string,
	description string,
	data *bytes.Buffer,
	contentType string,
	creatorID int64,
	isGroup bool,
) (*http.Request, error) {
	payload := createAssetRequest{
		AssetType:   assetType,
		DisplayName: name,
		Description: description,
	}
	if isGroup {
		payload.CreationContext.Creator.GroupID = creatorID
	} else {
		payload.CreationContext.Creator.UserID = creatorID
	}

	requestJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	formDataContentType := writer.FormDataContentType()

	go func() {
		defer pw.Close()

		field, err := writer.CreateFormField("request")
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := field.Write(requestJSON); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		fileHeader := make(textproto.MIMEHeader)
		fileHeader.Set("Content-Disposition", `form-data; name="fileContent"; filename="asset"`)
		fileHeader.Set("Content-Type", contentType)

		filePart, err := writer.CreatePart(fileHeader)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(filePart, bytes.NewReader(data.Bytes())); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if err := writer.Close(); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequest("POST", createAssetURL, pr)
	if err != nil {
		_ = pr.Close()
		return nil, err
	}
	req.Header.Set("User-Agent", "RobloxStudio/WinInet")
	req.Header.Set("Content-Type", formDataContentType)
	return req, nil
}

func decodeStatus(body []byte, fallback string) string {
	var status statusResponse
	if err := json.Unmarshal(body, &status); err == nil && status.Message != "" {
		return status.Message
	}
	return fallback
}

func isInappropriateError(message string) bool {
	lowered := strings.ToLower(message)
	return strings.Contains(lowered, "inappropriate name or description") || strings.Contains(lowered, "moderated")
}

func extractOperationID(path string) string {
	return strings.TrimPrefix(path, "operations/")
}

func parseAssetID(op *operationResponse) (int64, error) {
	if op == nil || op.Response == nil {
		return 0, errors.New("operation response is missing asset data")
	}

	if op.Response.AssetID != "" {
		id, err := strconv.ParseInt(op.Response.AssetID, 10, 64)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	path := strings.TrimSpace(op.Response.Path)
	if path == "" {
		return 0, errors.New("operation response is missing asset id")
	}

	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 || lastSlash == len(path)-1 {
		return 0, errors.New("operation response returned invalid asset path")
	}
	return strconv.ParseInt(path[lastSlash+1:], 10, 64)
}

func pollOperation(c *roblox.Client, operationID string) (*operationResponse, error) {
	req, err := http.NewRequest("GET", operationBaseURL+operationID, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.AddCookie(&http.Cookie{
		Name:  ".ROBLOSECURITY",
		Value: c.Cookie,
	})
	req.Header.Set("x-csrf-token", c.GetToken())
	req.Header.Set("User-Agent", "RobloxStudio/WinInet")
	setAPIKeyHeader(req)

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var operation operationResponse
		if err := json.Unmarshal(body, &operation); err != nil {
			return nil, err
		}
		return &operation, nil
	case http.StatusForbidden:
		c.SetToken(resp.Header.Get("x-csrf-token"))
		return nil, errTokenInvalid
	default:
		return nil, errors.New(decodeStatus(body, resp.Status))
	}
}

func executeCreateAsset(
	c *roblox.Client,
	req *http.Request,
	onTokenInvalid error,
	onNotLoggedIn error,
) (int64, error) {
	req.AddCookie(&http.Cookie{
		Name:  ".ROBLOSECURITY",
		Value: c.Cookie,
	})
	req.Header.Set("x-csrf-token", c.GetToken())
	setAPIKeyHeader(req)

	resp, err := c.DoRequest(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var operation operationResponse
		if err := json.Unmarshal(body, &operation); err != nil {
			return 0, err
		}
		if operation.Error != nil {
			return 0, errors.New(operation.Error.Message)
		}
		if operation.Done {
			return parseAssetID(&operation)
		}

		operationID := extractOperationID(operation.Path)
		if operationID == "" {
			return 0, errors.New("create asset operation id is empty")
		}

		for i := 0; i < maxPollAttempts; i++ {
			time.Sleep(pollInterval)
			polled, err := pollOperation(c, operationID)
			if err != nil {
				if errors.Is(err, errTokenInvalid) {
					return 0, onTokenInvalid
				}
				return 0, err
			}
			if !polled.Done {
				continue
			}
			if polled.Error != nil {
				return 0, errors.New(polled.Error.Message)
			}
			return parseAssetID(polled)
		}
		return 0, errors.New("asset operation timed out")
	case http.StatusUnauthorized:
		return 0, onNotLoggedIn
	case http.StatusTooManyRequests:
		return 0, ErrRateLimited
	case http.StatusForbidden:
		c.SetToken(resp.Header.Get("x-csrf-token"))
		return 0, onTokenInvalid
	default:
		return 0, errors.New(decodeStatus(body, resp.Status))
	}
}
