package develop

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kartFr/Asset-Reuploader/internal/retry"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var TeamCreateSettingsErrors = struct {
	ErrInvalidUniverse     error
	ErrUnauthorized        error
	ErrAuthorizationDenied error
}{
	ErrInvalidUniverse:     errors.New("invalid universe"),
	ErrUnauthorized:        errors.New("unauthorized"),
	ErrAuthorizationDenied: errors.New("authorization denied"),
}

type TeamCreateSettingsResponse struct {
	IsEnabled bool `json:"isEnabled"`
}

func newTeamCreateSettingsRequest(universeID int64) (*http.Request, error) {
	url := fmt.Sprintf("https://develop.roblox.com/v1/universes/%d/teamcreate", universeID)
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func NewTeamCreateSettingsHandler(c *roblox.Client, universeID int64) (func() (TeamCreateSettingsResponse, error), error) {
	req, err := newTeamCreateSettingsRequest(universeID)
	if err != nil {
		return func() (TeamCreateSettingsResponse, error) { return TeamCreateSettingsResponse{}, nil }, err
	}

	return func() (TeamCreateSettingsResponse, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})

		resp, err := c.DoRequest(req)
		if err != nil {
			return TeamCreateSettingsResponse{}, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var settings TeamCreateSettingsResponse
			json.NewDecoder(resp.Body).Decode(&settings)

			return settings, nil
		case http.StatusBadRequest:
			return TeamCreateSettingsResponse{}, TeamCreateSettingsErrors.ErrInvalidUniverse
		case http.StatusUnauthorized:
			return TeamCreateSettingsResponse{}, TeamCreateSettingsErrors.ErrUnauthorized
		case http.StatusForbidden:
			return TeamCreateSettingsResponse{}, TeamCreateSettingsErrors.ErrAuthorizationDenied
		default:
			return TeamCreateSettingsResponse{}, errors.New(resp.Status)
		}
	}, nil
}

func TeamCreateSettings(c *roblox.Client, universeID int64) (TeamCreateSettingsResponse, error) {
	handler, err := NewTeamCreateSettingsHandler(c, universeID)
	if err != nil {
		return TeamCreateSettingsResponse{}, err
	}

	return retry.Do(
		retry.NewOptions(retry.Tries(3)),
		func(_ int) (TeamCreateSettingsResponse, error) {
			settings, err := handler()
			if err != nil {
				if err == TeamCreateSettingsErrors.ErrInvalidUniverse ||
					err == TeamCreateSettingsErrors.ErrUnauthorized ||
					err == TeamCreateSettingsErrors.ErrAuthorizationDenied {

					return TeamCreateSettingsResponse{}, &retry.ExitRetry{Err: err}
				}

				return TeamCreateSettingsResponse{}, &retry.ContinueRetry{Err: err}
			}

			return settings, nil
		},
	)
}
