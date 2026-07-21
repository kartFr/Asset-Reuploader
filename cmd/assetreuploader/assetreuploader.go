package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kartFr/Asset-Reuploader/internal/app/config"
	"github.com/kartFr/Asset-Reuploader/internal/color"
	"github.com/kartFr/Asset-Reuploader/internal/console"
	"github.com/kartFr/Asset-Reuploader/internal/files"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
)

var (
	cookieFile = config.Get("cookie_file")
	apiKeyFile = config.Get("api_key_file")
	port       = config.Get("port")
)

func main() {
	console.ClearScreen()

	fmt.Println("Authenticating cookie...")

	cookie, readErr := files.Read(cookieFile)
	cookie = strings.TrimSpace(cookie)

	c, clientErr := roblox.NewClient(cookie)
	console.ClearScreen()

	if readErr != nil || clientErr != nil {
		if readErr != nil && !os.IsNotExist(readErr) {
			color.Error.Println(readErr)
		}

		if clientErr != nil && cookie != "" {
			color.Error.Println(clientErr)
		}

		getCookie(c)
	}

	if err := files.Write(cookieFile, c.Cookie); err != nil {
		color.Error.Println("Failed to save cookie: ", err)
	}

	apiKey, _ := files.Read(apiKeyFile)
	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" {
		c.SetAPIKey(apiKey)
	}

	console.ClearScreen()
	if apiKey == "" {
		getAPIKey(c)
	}

	if err := files.Write(apiKeyFile, c.APIKey); err != nil {
		color.Error.Println("Failed to save api key: ", err)
	}

	color.Success.Println("localhost = true (port " + port + ")")
	fmt.Println("Failed uploads will be automatically retried.")

	if err := serve(c); err != nil {
		log.Fatal(err)
	}
}

func getCookie(c *roblox.Client) {
	for {
		i, err := console.LongInput("ROBLOSECURITY: ")
		console.ClearScreen()
		if err != nil {
			color.Error.Println(err)
			continue
		}

		fmt.Println("Authenticating cookie...")
		err = c.SetCookie(i)
		console.ClearScreen()
		if err != nil {
			color.Error.Println(err)
			continue
		}

		files.Write(cookieFile, i)
		break
	}
}

func getAPIKey(c *roblox.Client) {
	for {
		fmt.Println("An API key is required to reupload animations.")
		fmt.Println("Create one at https://create.roblox.com/dashboard/credentials")
		fmt.Println("Make sure to add 'assets' with Read and Write permissions.")

		i, err := console.Input("API Key: ")
		console.ClearScreen()
		if err != nil {
			color.Error.Println(err)
			continue
		}

		if i == "" {
			color.Error.Println("API key cannot be empty")
			continue
		}

		c.SetAPIKey(i)
		break
	}
}
