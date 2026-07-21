package opencloud

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

const assetsURL = "https://apis.roblox.com/assets/v1/assets"

var UploadAnimationErrors = struct {
	ErrNotAuthenticated    error
	ErrInvalidAPIKey       error
	ErrInappropriateName   error
	ErrAssetTypeNotEnabled error
	ErrRateLimited         error
}{
	ErrNotAuthenticated:    errors.New("not authenticated"),
	ErrInvalidAPIKey:       errors.New("invalid api key"),
	ErrInappropriateName:   errors.New("inappropriate name or description"),
	ErrAssetTypeNotEnabled: errors.New("asset type not enabled for this api key"),
	ErrRateLimited:         errors.New("rate limited by api"),
}

type createAssetRequest struct {
	AssetType       string          `json:"assetType"`
	DisplayName     string          `json:"displayName"`
	Description     string          `json:"description"`
	CreationContext creationContext `json:"creationContext"`
}

type creationContext struct {
	Creator creator `json:"creator"`
}

type creator struct {
	UserID  string `json:"userId,omitempty"`
	GroupID string `json:"groupId,omitempty"`
}

type createAssetResponse struct {
	Path string `json:"path"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newCreateAnimationRequest(
	userID int64,
	groupID int64,
	name,
	description string,
	data *bytes.Buffer,
) (*http.Request, error) {
	var req createAssetRequest
	if groupID > 0 {
		req = createAssetRequest{
			AssetType:   "Animation",
			DisplayName: name,
			Description: description,
			CreationContext: creationContext{
				Creator: creator{
					GroupID: strconv.FormatInt(groupID, 10),
				},
			},
		}
	} else {
		req = createAssetRequest{
			AssetType:   "Animation",
			DisplayName: name,
			Description: description,
			CreationContext: creationContext{
				Creator: creator{
					UserID: strconv.FormatInt(userID, 10),
				},
			},
		}
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add the "request" form field
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="request"`)
	h.Set("Content-Type", "application/json")
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(reqJSON); err != nil {
		return nil, err
	}

	// Add the "fileContent" form field with the animation data
	h = make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="fileContent"; filename="animation.rbxm"`)
	h.Set("Content-Type", "model/x-rbxm")
	part, err = writer.CreatePart(h)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", assetsURL, &body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	return httpReq, nil
}

func NewUploadAnimationHandler(
	c *roblox.Client,
	name,
	description string,
	data *bytes.Buffer,
	groupID ...int64,
) (func() (int64, error), error) {
	group := groupID[0]
	req, err := newCreateAnimationRequest(c.UserInfo.ID, group, name, description, data)
	if err != nil {
		return func() (int64, error) { return 0, nil }, err
	}

	// Save the body bytes and content type for retry
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return func() (int64, error) { return 0, nil }, err
	}
	req.Body.Close()
	contentType := req.Header.Get("Content-Type")

	return func() (int64, error) {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", contentType)

		resp, err := c.DoAPIKeyRequest(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		switch resp.StatusCode {
		case http.StatusOK:
			var createResp createAssetResponse
			if err := json.Unmarshal(respBody, &createResp); err != nil {
				return 0, err
			}

			operationID := createResp.Path
			if operationID == "" {
				return 0, errors.New("empty operation path returned")
			}

			assetID, err := GetOperation(c, operationID)
			if err != nil {
				return 0, err
			}
			return assetID, nil
		case http.StatusTooManyRequests:
			return 0, UploadAnimationErrors.ErrRateLimited
		case http.StatusForbidden, http.StatusUnauthorized:
			var errResp errorResponse
			if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil {
				if errResp.Code == "PERMISSION_DENIED" || errResp.Code == "UNAUTHENTICATED" {
					return 0, UploadAnimationErrors.ErrInvalidAPIKey
				}
			}
			return 0, UploadAnimationErrors.ErrNotAuthenticated
		case http.StatusBadRequest:
			var errResp errorResponse
			if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil {
				switch errResp.Code {
				case "INVALID_ARGUMENT":
					if errResp.Message == "Inappropriate name or description." {
						return 0, UploadAnimationErrors.ErrInappropriateName
					}
				case "ASSET_TYPE_NOT_ENABLED":
					return 0, UploadAnimationErrors.ErrAssetTypeNotEnabled
				}
			}
			return 0, fmt.Errorf("%s: %s", resp.Status, string(respBody))
		default:
			return 0, errors.New(resp.Status)
		}
	}, nil
}
