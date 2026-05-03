// assetreuploader.go — PATCH: add ensureAPIKey() call and function
//
// In main(), after saving the cookie (around line 44), add:
//     ensureAPIKey()
//
// Then add the following function at the bottom of the file:

package main

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
	"github.com/kartFr/Asset-Reuploader/internal/app/config"
	"github.com/kartFr/Asset-Reuploader/internal/console"
)

// ensureAPIKey prompts the user to enter an Open Cloud API key if none is
// configured. The key is required for animation uploads since Roblox deprecated
// the old UploadNewAnimation endpoint (HTTP 410 as of ~April 2026).
func ensureAPIKey() {
	if strings.TrimSpace(config.Get("api_key")) != "" {
		return // already configured
	}

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("  Open Cloud API Key Required for Animation Uploads")
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("Roblox deprecated the old animation upload endpoint.")
	fmt.Println("You now need a free Open Cloud API key.")
	fmt.Println()
	fmt.Println("How to get one (takes ~1 minute):")
	fmt.Println("  1. Go to https://create.roblox.com/dashboard/credentials?activeTab=ApiKeysTab")
	fmt.Println("  2. Click  Create API Key")
	fmt.Println("  3. Enter any name (e.g. \"Asset Reuploader\")")
	fmt.Println("  4. Under  Select API System  choose  Assets")
	fmt.Println("  5. Enable  Write  permission for Assets")
	fmt.Println("  6. Click  Save  and copy the key")
	fmt.Println()

	key, err := console.Input("Paste your API key here (leave blank to skip): ")
	if err != nil {
		color.Error.Println(err)
		return
	}

	key = strings.TrimSpace(key)
	if key == "" {
		fmt.Println("Skipping — animation reuploads will fail until api_key is set in config.ini")
		return
	}

	config.Set("api_key", key)
	if err := config.Save(); err != nil {
		color.Error.Println("Failed to save api_key to config.ini:", err)
		return
	}
	fmt.Println("API key saved to config.ini ✓")
	fmt.Println()
}
