package roblox

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kartFr/Asset-Reuploader/internal/retry"
)

var AuthenticateErrors = struct {
	ErrAuthorizationDenied error
}{
	ErrAuthorizationDenied: errors.New("invalid cookie"),
}

type UserInfo struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

func authenticateHandler(c *Client, cookie string) (func() (UserInfo, error), error) {
	req, err := http.NewRequest("GET", "https://users.roblox.com/v1/users/authenticated", http.NoBody)
	if err != nil {
		return func() (UserInfo, error) { return UserInfo{}, nil }, err
	}
	req.AddCookie(&http.Cookie{
		Name:  ".ROBLOSECURITY",
		Value: cookie,
	})

	return func() (UserInfo, error) {
		resp, err := c.DoRequest(req)
		if err != nil {
			return UserInfo{}, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var userInfo UserInfo
			json.NewDecoder(resp.Body).Decode(&userInfo)

			return userInfo, nil
		case http.StatusUnauthorized:
			return UserInfo{}, AuthenticateErrors.ErrAuthorizationDenied
		default:
			return UserInfo{}, errors.New(resp.Status)
		}
	}, nil
}

func authenticate(c *Client, cookie string) (UserInfo, error) {
	handler, err := authenticateHandler(c, cookie)
	if err != nil {
		return UserInfo{}, nil
	}

	userInfo, err := retry.Do(
		retry.NewOptions(retry.Tries(3)),
		func(_ int) (UserInfo, error) {
			userInfo, err := handler()
			if err != nil {
				if err == AuthenticateErrors.ErrAuthorizationDenied {
					return UserInfo{}, &retry.ExitRetry{Err: err}
				}

				return UserInfo{}, &retry.ContinueRetry{Err: err}
			}

			return userInfo, nil
		},
	)
	return userInfo, err
}
