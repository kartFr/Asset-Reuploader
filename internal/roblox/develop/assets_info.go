package develop

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

type AssetInfo struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	TypeID      int32  `json:"typeId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Creator     struct {
		Type     string `json:"type"`
		TypeID   int32  `json:"typeId"`
		TargetID int64  `json:"targetId"`
	} `json:"creator"`
	Genres                []string  `json:"genres"`
	Created               time.Time `json:"created"`
	Updated               time.Time `json:"updated"`
	EnableComments        bool      `json:"enableComments"`
	IsCopyingAllowed      bool      `json:"isCopyingAllowed"`
	IsPublicDomainEnabled bool      `json:"isPublicDomainEnabled"`
	IsModerated           bool      `json:"isModerated"`
	ReviewStatus          string    `json:"reviewStatus"`
	IsVersioningEnabled   bool      `json:"isVersioningEnabled"`
	IsArchivable          bool      `json:"isArchivable"`
	CanHaveThumbnail      bool      `json:"canHaveThumbnail"`
}

var GetAssetsInfoErrors = struct {
	ErrUnauthorized error
}{
	ErrUnauthorized: errors.New("unauthorized"),
}

type GetAssetsInfoResponse struct {
	Data   []*AssetInfo `json:"data,omitempty"`
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func newAssetInfoBulkURL(assetIDs []int64) string {
	strIDs := make([]string, len(assetIDs))
	for i, id := range assetIDs {
		strIDs[i] = strconv.FormatInt(id, 10)
	}

	return fmt.Sprintf("https://develop.roblox.com/v1/assets?assetIds=%s", strings.Join(strIDs, ","))
}

func newAssetsInfoRequest(assetIDs []int64) (*http.Request, error) {
	url := newAssetInfoBulkURL(assetIDs)
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func NewAssetsInfoHandler(c *roblox.Client, assetIDs []int64) (func() (GetAssetsInfoResponse, error), error) {
	req, err := newAssetsInfoRequest(assetIDs)
	if err != nil {
		return func() (GetAssetsInfoResponse, error) { return GetAssetsInfoResponse{}, nil }, err
	}

	return func() (GetAssetsInfoResponse, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})

		resp, err := c.DoRequest(req)
		if err != nil {
			return GetAssetsInfoResponse{}, err
		}
		defer resp.Body.Close()

		var bulkResponse GetAssetsInfoResponse
		json.NewDecoder(resp.Body).Decode(&bulkResponse)

		switch resp.StatusCode {
		case http.StatusOK:
			return bulkResponse, nil
		case http.StatusUnauthorized:
			return bulkResponse, GetAssetsInfoErrors.ErrUnauthorized
		default:
			if bulkResponse.Errors != nil {
				if message := bulkResponse.Errors[0].Message; message != "" {
					return bulkResponse, errors.New(bulkResponse.Errors[0].Message)
				}
			}

			return bulkResponse, errors.New(resp.Status)
		}
	}, nil
}
