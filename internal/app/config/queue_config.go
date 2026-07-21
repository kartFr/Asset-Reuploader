package config

import (
	"encoding/json"
	"os"
	"time"
)

type QueueConfig struct {
	Window string `json:"window"`
	Limit  int    `json:"limit"`
}

type Config struct {
	UploadQueue     QueueConfig `json:"uploadQueue"`
	GroupGameQueue  QueueConfig `json:"groupGameQueue"`
	UserGameQueue   QueueConfig `json:"userGameQueue"`
}

var queueConfig Config

func init() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return
	}

	if err := json.Unmarshal(data, &queueConfig); err != nil {
		return
	}
}

func GetUploadQueueConfig() (time.Duration, int) {
	window, err := time.ParseDuration(queueConfig.UploadQueue.Window)
	if err != nil {
		return time.Minute, 60
	}
	return window, queueConfig.UploadQueue.Limit
}

func GetGroupGameQueueConfig() (time.Duration, int) {
	window, err := time.ParseDuration(queueConfig.GroupGameQueue.Window)
	if err != nil {
		return 5 * time.Second, 5
	}
	return window, queueConfig.GroupGameQueue.Limit
}

func GetUserGameQueueConfig() (time.Duration, int) {
	window, err := time.ParseDuration(queueConfig.UserGameQueue.Window)
	if err != nil {
		return 5 * time.Second, 5
	}
	return window, queueConfig.UserGameQueue.Limit
}
