package permissions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var UpdatePermissionErrors = struct {
	ErrTokenInvalid     error
	ErrNotAuthenticated error
	ErrUnsupportedAsset error
}{
	ErrTokenInvalid:     errors.New("XSRF token validation failed"),
	ErrNotAuthenticated: errors.New("user is not authenticated"),
	ErrUnsupportedAsset: errors.New("asset cannot be modified via permissions endpoint"),
}

type PermissionRequestItem struct {
	SubjectType string `json:"subjectType"`
	SubjectID   string `json:"subjectId"`
	Action      string `json:"action"`
}

type PermissionRequest struct {
	Requests []PermissionRequestItem `json:"requests"`
}

type PermissionResponse struct {
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func newUpdatePermissionsRequest(assetID int64, body PermissionRequest) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://apis.roblox.com/asset-permissions-api/v1/assets/%d/permissions", assetID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func NewUpdatePermissionsHandler(c *roblox.Client, assetID int64, body PermissionRequest) (func() (*PermissionResponse, error), error) {
	req, err := newUpdatePermissionsRequest(assetID, body)
	if err != nil {
		return func() (*PermissionResponse, error) { return nil, nil }, err
	}

	return func() (*PermissionResponse, error) {
		for attempt := 0; attempt < 2; attempt++ {
			req2 := req.Clone(req.Context())

			req2.AddCookie(&http.Cookie{
				Name:  ".ROBLOSECURITY",
				Value: c.Cookie,
			})

			if token := c.GetToken(); token != "" {
				req2.Header.Set("x-csrf-token", token)
			}

			resp, err := c.DoRequest(req2)
			if err != nil {
				return nil, err
			}

			defer resp.Body.Close()

			var response PermissionResponse
			_ = json.NewDecoder(resp.Body).Decode(&response)

			if resp.StatusCode == http.StatusOK {
				return &response, nil
			}

			if resp.StatusCode == http.StatusForbidden {
				if token := resp.Header.Get("x-csrf-token"); token != "" {
					c.SetToken(token)
					continue
				}
				return nil, UpdatePermissionErrors.ErrTokenInvalid
			}

			if resp.StatusCode == http.StatusUnauthorized {
				return nil, UpdatePermissionErrors.ErrNotAuthenticated
			}

			if resp.StatusCode == http.StatusGone {
				return nil, UpdatePermissionErrors.ErrUnsupportedAsset
			}

			if response.Errors != nil && len(response.Errors) > 0 {
				return nil, errors.New(response.Errors[0].Message)
			}

			return nil, errors.New(resp.Status)
		}

		return nil, UpdatePermissionErrors.ErrTokenInvalid
	}, nil
}
