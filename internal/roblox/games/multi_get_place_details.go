package games

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kartFr/Asset-Reuploader/internal/retry"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var MultiGetPlaceDetailsErrors = struct {
	ErrUnauthorized error
}{
	ErrUnauthorized: errors.New("unauthorized"),
}

type PlaceDetailsResponse struct {
	PlaceID             int64  `json:"placeId"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	SourceName          string `json:"sourceName"`
	SourceDescription   string `json:"sourceDescription"`
	URL                 string `json:"url"`
	Builder             string `json:"builder"`
	BuilderID           int64  `json:"builderId"`
	HasVerifiedBadge    bool   `json:"hasVerifiedBadge"`
	IsPlayable          bool   `json:"isPlayable"`
	ReasonProhibited    string `json:"reasonProhibited"`
	UniverseID          int64  `json:"universeId"`
	UniverseRootPlaceID int64  `json:"universeRootPlaceId"`
	Price               int64  `json:"price"`
	ImageToken          string `json:"imageToken"`
	FiatPurchaseData    struct {
		LocalizedFiatPrice string `json:"localizedFiatPrice"`
		BasePriceID        string `json:"basePriceId"`
	} `json:"fiatPurchaseData"`
}

func newMultiGetPlaceURL(placeIDs []int64) string {
	strIDs := make([]string, len(placeIDs))
	for i, id := range placeIDs {
		strIDs[i] = strconv.FormatInt(id, 10)
	}

	return fmt.Sprintf("https://games.roblox.com/v1/games/multiget-place-details?placeIds=%s", strings.Join(strIDs, ","))
}

func NewMultiGetPlaceDetailsHandler(c *roblox.Client, placeIDs []int64) (func() ([]*PlaceDetailsResponse, error), error) {
	url := newMultiGetPlaceURL(placeIDs)
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return func() ([]*PlaceDetailsResponse, error) { return make([]*PlaceDetailsResponse, 0), nil }, err
	}

	return func() ([]*PlaceDetailsResponse, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})

		resp, err := c.DoRequest(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var placeDetails []*PlaceDetailsResponse
			json.NewDecoder(resp.Body).Decode(&placeDetails)
			return placeDetails, nil
		case http.StatusUnauthorized:
			return nil, MultiGetPlaceDetailsErrors.ErrUnauthorized
		default:
			return nil, errors.New(resp.Status)
		}
	}, nil
}

func MultiGetPlaceDetails(c *roblox.Client, placeIDs []int64) ([]*PlaceDetailsResponse, error) {
	handler, err := NewMultiGetPlaceDetailsHandler(c, placeIDs)
	if err != nil {
		return nil, err
	}

	return retry.Do(
		retry.NewOptions(retry.Tries(3)),
		func(_ int) ([]*PlaceDetailsResponse, error) {
			placeDetails, err := handler()
			if err != nil {
				if err == MultiGetPlaceDetailsErrors.ErrUnauthorized {
					return nil, &retry.ExitRetry{Err: err}
				}

				return nil, &retry.ContinueRetry{Err: err}
			}

			return placeDetails, nil
		},
	)
}
