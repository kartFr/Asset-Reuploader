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
	ensureAPIKey()

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

func ensureAPIKey() {
	if strings.TrimSpace(config.Get("api_key")) != "" {
		return
	}

	fmt.Println("Enter your Open Cloud API key to enable mesh/animation uploads.")
	fmt.Println("How to get one:")
	fmt.Println("1. Go to https://create.roblox.com/dashboard/credentials?activeTab=ApiKeysTab")
	fmt.Println("2. Click Create API Key")
	fmt.Println("3. Enter any name")
	fmt.Println("4. Select Assets in Select API System")
	fmt.Println("5. Select Write in each Assets permission")
	key, err := console.Input("API key (leave blank to skip): ")
	if err != nil {
		color.Error.Println(err)
		return
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	config.Set("api_key", key)
	if err := config.Save(); err != nil {
		color.Error.Println("Failed to save api key: ", err)
	}
}
