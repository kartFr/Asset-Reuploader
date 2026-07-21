package opencloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

const operationsURL = "https://apis.roblox.com/assets/v1/"

var (
	ErrOperationNotDone = errors.New("operation not done yet")
	ErrOperationFailed  = errors.New("operation failed")
)

type operationResponse struct {
	Path     string          `json:"path"`
	Done     bool           `json:"done"`
	Response *operationResult `json:"response,omitempty"`
	Error    *operationError  `json:"error,omitempty"`
}

type operationResult struct {
	AssetID string `json:"assetId"`
}

type operationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func GetOperation(c *roblox.Client, operationPath string) (int64, error) {
	url := operationsURL + operationPath

	for i := 0; i < 30; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return 0, err
		}

		resp, err := c.DoAPIKeyRequest(req)
		if err != nil {
			return 0, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return 0, err
		}

		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("get operation failed: %s", resp.Status)
		}

		var opResp operationResponse
		if err := json.Unmarshal(body, &opResp); err != nil {
			return 0, err
		}

		if !opResp.Done {
			time.Sleep(time.Second)
			continue
		}

		if opResp.Error != nil {
			return 0, fmt.Errorf("%s: %s", opResp.Error.Code, opResp.Error.Message)
		}

		if opResp.Response == nil {
			return 0, ErrOperationFailed
		}

		assetID, err := strconv.ParseInt(opResp.Response.AssetID, 10, 64)
		if err != nil {
			return 0, err
		}

		return assetID, nil
	}

	return 0, ErrOperationNotDone
}
