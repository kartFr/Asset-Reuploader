package ide

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var UploadAnimationErrors = struct {
	ErrNotLoggedIn       error
	ErrTokenInvalid      error
	ErrInappropriateName error
	ErrBadResponse       error
}{
	ErrNotLoggedIn:       errors.New("not logged in"),
	ErrTokenInvalid:      errors.New("XSRF token validation failed"),
	ErrInappropriateName: errors.New("inappropriate name or description"),
	ErrBadResponse:       errors.New("invalid upload response"),
}

func newAnimationURL(groupID int64, name, description string) string {
	u := fmt.Sprintf(
		"https://www.roblox.com/ide/publish/UploadNewAnimation?assetTypeName=Animation&name=%s&description=%s",
		url.QueryEscape(name),
		url.QueryEscape(description),
	)
	if groupID > 0 {
		u += fmt.Sprintf("&groupId=%d", groupID)
	}

	return u
}

func newUploadAnimationRequest(
	groupID int64,
	name,
	description string,
	data *bytes.Buffer,
) (*http.Request, error) {
	u := newAnimationURL(groupID, name, description)

	var body io.Reader = http.NoBody
	if data != nil {
		body = bytes.NewReader(data.Bytes())
	}

	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "RobloxStudio/WinInet")
	req.Header.Set("Accept", "text/plain, application/json, */*")

	return req, nil
}

func NewUploadAnimationHandler(
	c *roblox.Client,
	name,
	description string,
	data *bytes.Buffer,
	groupID ...int64,
) (func() (int64, error), error) {
	var group int64
	if len(groupID) > 0 {
		group = groupID[0]
	}

	if _, err := newUploadAnimationRequest(group, name, description, data); err != nil {
		return func() (int64, error) { return 0, nil }, err
	}

	return func() (int64, error) {
		currentName := name

		for attempt := 0; attempt < 2; attempt++ {
			req, err := newUploadAnimationRequest(group, currentName, description, data)
			if err != nil {
				return 0, err
			}

			req.AddCookie(&http.Cookie{
				Name:  ".ROBLOSECURITY",
				Value: c.Cookie,
			})

			if token := c.GetToken(); token != "" {
				req.Header.Set("x-csrf-token", token)
			}

			resp, err := c.DoRequest(req)
			if err != nil {
				return 0, err
			}

			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return 0, readErr
			}

			strBody := strings.TrimSpace(string(body))

			switch resp.StatusCode {
			case http.StatusOK:
				id, err := strconv.ParseInt(strBody, 10, 64)
				if err != nil || id <= 0 {
					return 0, UploadAnimationErrors.ErrBadResponse
				}
				return id, nil

			case http.StatusForbidden:
				lowerBody := strings.ToLower(strBody)

				if strBody == "NotLoggedIn" || strings.Contains(lowerBody, "notloggedin") {
					return 0, UploadAnimationErrors.ErrNotLoggedIn
				}

				if strBody == "XSRF Token Validation Failed" || strings.Contains(lowerBody, "xsrf") {
					if token := resp.Header.Get("x-csrf-token"); token != "" {
						c.SetToken(token)
						if attempt == 0 {
							continue
						}
					}
					return 0, UploadAnimationErrors.ErrTokenInvalid
				}

				if strBody != "" {
					return 0, errors.New(strBody)
				}

				return 0, errors.New(resp.Status)

			case http.StatusUnauthorized:
				return 0, UploadAnimationErrors.ErrNotLoggedIn

			case http.StatusUnprocessableEntity:
				if strBody == "Inappropriate name or description." || strings.Contains(strings.ToLower(strBody), "inappropriate") {
					currentName = "[Censored]"
					return 0, UploadAnimationErrors.ErrInappropriateName
				}

				if strBody != "" {
					return 0, errors.New(strBody)
				}

				return 0, errors.New(resp.Status)

			case http.StatusTooManyRequests:
				if strBody != "" {
					return 0, errors.New(strBody)
				}
				return 0, errors.New(resp.Status)

			default:
				if strBody != "" {
					return 0, errors.New(strBody)
				}
				return 0, errors.New(resp.Status)
			}
		}

		return 0, UploadAnimationErrors.ErrTokenInvalid
	}, nil
}
