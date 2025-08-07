package games

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kartFr/Asset-Reuploader/internal/retry"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

func NewUserGamesHandler(c *roblox.Client, userID int64) (func() (*GamesResponse, error), error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://games.roblox.com/v2/users/%d/games?limit=50", userID), http.NoBody)
	if err != nil {
		return func() (*GamesResponse, error) { return nil, nil }, err
	}

	return func() (*GamesResponse, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})

		resp, err := c.DoRequest(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(resp.Status)
		}

		var gamesResponse GamesResponse
		json.NewDecoder(resp.Body).Decode(&gamesResponse)
		return &gamesResponse, nil
	}, nil
}

func UserGames(c *roblox.Client, userID int64) (*GamesResponse, error) {
	handler, err := NewUserGamesHandler(c, userID)
	if err != nil {
		return nil, err
	}

	return retry.Do(
		retry.NewOptions(retry.Tries(3)),
		func(_ int) (*GamesResponse, error) {
			placeDetails, err := handler()
			if err != nil {
				return nil, &retry.ContinueRetry{Err: err}
			}

			return placeDetails, nil
		},
	)
}
