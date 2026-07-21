package clientutils

import (
	"errors"
	"fmt"

	"github.com/kartFr/Asset-Reuploader/internal/app/config"
	"github.com/kartFr/Asset-Reuploader/internal/app/context"
	"github.com/kartFr/Asset-Reuploader/internal/color"
	"github.com/kartFr/Asset-Reuploader/internal/console"
	"github.com/kartFr/Asset-Reuploader/internal/files"
)

var apiKeyFile = config.Get("api_key_file")

func GetNewAPIKey(ctx *context.Context, m string) {
	pauseController := ctx.PauseController

	if !pauseController.Pause() {
		pauseController.WaitIfPaused()
		return
	}

	console.ClearScreen()

	client := ctx.Client
	inputErr := errors.New(m)
	for {
		fmt.Print(ctx.Logger.History.String())
		color.Error.Println(inputErr)

		i, err := console.Input("API Key: ")
		console.ClearScreen()
		if err != nil {
			inputErr = err
			continue
		}

		if i == "" {
			inputErr = errors.New("api key cannot be empty")
			continue
		}

		client.SetAPIKey(i)
		break
	}

	fmt.Print(ctx.Logger.History.String())

	if err := files.Write(apiKeyFile, client.APIKey); err != nil {
		ctx.Logger.Error("Failed to save api key: ", err)
	}

	pauseController.Unpause()
}
