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
	if strings.TrimSpace(config.Get("api_key")) == "" {
		getAPIKey()
	}
	fmt.Println("localhost started on port " + port + ". Waiting to start reuploading.")
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

func getAPIKey() {
	for {
		fmt.Println("An Open Cloud API Key is required for animation uploads.")
		fmt.Println("Get one at: https://create.roblox.com/dashboard/credentials?activeTab=ApiKeysTab")
		fmt.Println("(Create API Key -> Assets -> Write permission)")
		i, err := console.LongInput("API Key: ")
		console.ClearScreen()
		if err != nil {
			color.Error.Println(err)
			continue
		}
		i = strings.TrimSpace(i)
		if i == "" {
			color.Error.Println("API Key cannot be empty")
			continue
		}
		config.Set("api_key", i)
		if err := config.Save(); err != nil {
			color.Error.Println("Failed to save API Key: ", err)
		}
		break
	}
}