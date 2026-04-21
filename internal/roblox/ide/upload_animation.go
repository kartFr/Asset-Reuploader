package ide

import (
	"bytes"
	"errors"

	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var UploadAnimationErrors = struct {
	ErrNotLoggedIn       error
	ErrTokenInvalid      error
	ErrInappropriateName error
}{
	ErrNotLoggedIn:       errors.New("not logged in"),
	ErrTokenInvalid:      errors.New("XSRF token validation failed"),
	ErrInappropriateName: errors.New("inappropriate name or description"),
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
	currentName := name

	return func() (int64, error) {
		req, err := newCreateAssetRequest(
			"Animation",
			currentName,
			description,
			data,
			"model/x-rbxm",
			func() int64 {
				if group > 0 {
					return group
				}
				return c.UserInfo.ID
			}(),
			group > 0,
		)
		if err != nil {
			return 0, err
		}

		id, err := executeCreateAsset(c, req, UploadAnimationErrors.ErrTokenInvalid, UploadAnimationErrors.ErrNotLoggedIn)
		if err == nil {
			return id, nil
		}

		if isInappropriateError(err.Error()) {
			currentName = "[Censored]"
			return 0, UploadAnimationErrors.ErrInappropriateName
		}

		return 0, err
	}, nil
}
