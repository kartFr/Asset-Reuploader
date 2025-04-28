package assets

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
}{
	ErrTokenInvalid:     errors.New("XSRF token validation failed"),
	ErrNotAuthenticated: errors.New("user is not authenticated"),
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

func newUpdatePermissionsRequest(assetId int64, body PermissionRequest) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://apis.roblox.com/asset-permissions-api/v1/assets/%d/permissions", assetId)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(jsonBody))
	if err != nil {
		return req, err
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func NewUpdatePermissionsHandler(c *roblox.Client, assetId int64, body PermissionRequest) (func() (*PermissionResponse, error), error) {
	req, err := newUpdatePermissionsRequest(assetId, body)
	if err != nil {
		return func() (*PermissionResponse, error) { return nil, nil }, err
	}

	return func() (*PermissionResponse, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})
		req.Header.Set("x-csrf-token", c.GetToken())

		resp, err := c.DoRequest(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var response PermissionResponse
		json.NewDecoder(resp.Body).Decode(&response)

		switch resp.StatusCode {
		case http.StatusOK:
			return &response, nil
		case http.StatusUnauthorized:
			return nil, UpdatePermissionErrors.ErrNotAuthenticated
		case http.StatusForbidden:
			c.SetToken(resp.Header.Get("x-csrf-token"))
			return nil, UpdatePermissionErrors.ErrTokenInvalid
		default:
			if response.Errors != nil {
				if message := response.Errors[0].Message; message != "" {
					return nil, errors.New(response.Errors[0].Message)
				}
			}

			return nil, errors.New(resp.Status)
		}
	}, nil
}
