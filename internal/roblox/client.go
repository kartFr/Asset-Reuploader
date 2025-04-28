package roblox

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const cookieWarning = "WARNING:-DO-NOT-SHARE-THIS.--Sharing-this-will-allow-someone-to-log-in-as-you-and-to-steal-your-ROBUX-and-items."

var (
	ErrNoWarning = errors.New("include the .ROBLOSECURITY warning")
)

type Client struct {
	Cookie   string
	UserInfo UserInfo

	httpClient *http.Client

	token      string
	tokenMutex sync.RWMutex
}

func NewClient(cookie string) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if err := c.SetCookie(cookie); err != nil {
		return c, err
	}

	return c, nil
}

func (c *Client) SetCookie(cookie string) error {
	c.Cookie = strings.TrimSpace(cookie)

	if !strings.Contains(cookie, cookieWarning) {
		return ErrNoWarning
	}

	userInfo, err := authenticate(c, cookie)
	if err != nil {
		return err
	}

	c.UserInfo = userInfo
	c.Cookie = cookie
	return nil
}

func (c *Client) GetToken() string {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.token
}

func (c *Client) SetToken(s string) {
	c.tokenMutex.Lock()
	c.token = s
	c.tokenMutex.Unlock()
}

func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}
