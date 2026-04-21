package roblox

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/kartFr/Asset-Reuploader/internal/retry"
)

var AuthenticateErrors = struct {
	ErrAuthorizationDenied error
	ErrBadResponse         error
}{
	ErrAuthorizationDenied: errors.New("invalid cookie"),
	ErrBadResponse:         errors.New("invalid roblox response"),
}

type UserInfo struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type authenticateErrorResponse struct {
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func newAuthenticateRequest(cookie string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, "https://users.roblox.com/v1/users/authenticated", http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  ".ROBLOSECURITY",
		Value: cookie,
	})

	return req, nil
}

func authenticateHandler(c *Client, cookie string) (func() (UserInfo, error), error) {
	if _, err := newAuthenticateRequest(cookie); err != nil {
		return func() (UserInfo, error) { return UserInfo{}, nil }, err
	}

	return func() (UserInfo, error) {
		req, err := newAuthenticateRequest(cookie)
		if err != nil {
			return UserInfo{}, err
		}

		resp, err := c.DoRequest(req)
		if err != nil {
			return UserInfo{}, err
		}
		defer resp.Body.Close()

		rawBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return UserInfo{}, err
		}

		switch resp.StatusCode {
		case http.StatusOK:
			var userInfo UserInfo
			if err := json.Unmarshal(rawBody, &userInfo); err != nil {
				return UserInfo{}, AuthenticateErrors.ErrBadResponse
			}
			if userInfo.ID == 0 {
				return UserInfo{}, AuthenticateErrors.ErrBadResponse
			}
			return userInfo, nil

		case http.StatusUnauthorized:
			return UserInfo{}, AuthenticateErrors.ErrAuthorizationDenied

		default:
			var errorResponse authenticateErrorResponse
			if err := json.Unmarshal(rawBody, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
				msg := strings.TrimSpace(errorResponse.Errors[0].Message)
				if msg != "" {
					return UserInfo{}, errors.New(msg)
				}
			}

			bodyText := strings.TrimSpace(string(rawBody))
			if bodyText != "" {
				return UserInfo{}, errors.New(bodyText)
			}

			return UserInfo{}, errors.New(resp.Status)
		}
	}, nil
}

func authenticate(c *Client, cookie string) (UserInfo, error) {
	handler, err := authenticateHandler(c, cookie)
	if err != nil {
		return UserInfo{}, err
	}

	userInfo, err := retry.Do(
		retry.NewOptions(retry.Tries(3)),
		func(_ int) (UserInfo, error) {
			userInfo, err := handler()
			if err != nil {
				if errors.Is(err, AuthenticateErrors.ErrAuthorizationDenied) {
					return UserInfo{}, &retry.ExitRetry{Err: err}
				}

				if errors.Is(err, AuthenticateErrors.ErrBadResponse) {
					return UserInfo{}, &retry.ExitRetry{Err: err}
				}

				return UserInfo{}, &retry.ContinueRetry{Err: err}
			}

			return userInfo, nil
		},
	)

	return userInfo, err
}
