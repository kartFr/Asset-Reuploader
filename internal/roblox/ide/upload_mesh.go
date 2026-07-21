package ide

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var once sync.Once

var (
	uploadURL  string
	queryStuff string
)

var UploadMeshErrors = struct {
	ErrNotLoggedIn       error
	ErrTokenInvalid      error
	ErrInappropriateName error
	ErrRateLimited       error
}{
	ErrNotLoggedIn:       errors.New("not logged in"),
	ErrTokenInvalid:      errors.New("XSRF token validation failed"),
	ErrInappropriateName: errors.New("inappropriate name or description"),
	ErrRateLimited:       errors.New("rate limited by api"),
}

func xorBytes(data []byte, key byte) []byte {
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ key
	}
	return out
}

func newUploadMeshURL(groupID int64, name, description string) string {
	url := fmt.Sprintf("https://data.roblox.com/ide/publish/UploadNewMesh?assetTypeName=Mesh&name=%s&description=%s"+queryStuff,
		url.QueryEscape(name),
		url.QueryEscape(description),
	)
	if groupID > 0 {
		url += fmt.Sprintf("&groupId=%d", groupID)
	}

	return url
}

func newUploadMeshRequest(
	groupID int64,
	name,
	description string,
	data *bytes.Buffer,
) (*http.Request, error) {
	url := newUploadMeshURL(groupID, name, description)
	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RobloxStudio/WinInet")

	return req, nil
}

func NewUploadMeshHandler(
	c *roblox.Client,
	name,
	description string,
	data *bytes.Buffer,
	groupID ...int64,
) (func() (int64, error), error) {
	group := groupID[0]
	req, err := newUploadMeshRequest(group, name, description, data)
	if err != nil {
		return func() (int64, error) { return 0, nil }, err
	}

	return func() (int64, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})
		req.Header.Set("x-csrf-token", c.GetToken())

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
			id, err := strconv.ParseInt(string(body), 10, 64)
			if err != nil {
				return 0, err
			}

			return id, nil
		case http.StatusTooManyRequests:
			return 0, UploadMeshErrors.ErrRateLimited
		case http.StatusForbidden:
			if strBody := string(body); strBody == "NotLoggedIn" {
				return 0, UploadMeshErrors.ErrNotLoggedIn
			} else if strBody == "XSRF Token Validation Failed" {
				c.SetToken(resp.Header.Get("x-csrf-token"))
				return 0, UploadMeshErrors.ErrTokenInvalid
			}

			return 0, errors.New(resp.Status)
		case http.StatusUnprocessableEntity:
			if string(body) == "Inappropriate name or description." {
				req, _ = newUploadMeshRequest(group, "[Censored]", description, data)
				return 0, UploadMeshErrors.ErrInappropriateName
			}

			return 0, errors.New(resp.Status)
		default:
			return 0, errors.New(resp.Status)
		}
	}, nil
}
