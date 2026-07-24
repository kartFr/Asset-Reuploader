package roblox

import (
	"errors"
)

var (
	ErrNoAPIKey = errors.New("no API key provided")
)

// OpenCloudAuth represents authentication for Open Cloud API
type OpenCloudAuth struct {
	APIKey string
}

// NewOpenCloudAuth creates a new Open Cloud authentication instance
func NewOpenCloudAuth(apiKey string) (*OpenCloudAuth, error) {
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}

	return &OpenCloudAuth{
		APIKey: apiKey,
	}, nil
}
