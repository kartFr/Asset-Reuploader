# Open Cloud API Migration Guide

## Overview

Roblox has deprecated the legacy IDE endpoints for asset uploads (returning 410 Gone errors). This guide explains how to migrate Asset Reuploader to use the **Open Cloud API**.

## What Changed?

- **Old System**: Used `.ROBLOSECURITY` cookies + IDE endpoints (`/ide/publish/UploadNewAnimation`)
- **New System**: Uses Open Cloud API keys + modern REST endpoints

## Why Migrate?

✅ **Officially Supported**: Open Cloud API is Roblox's modern approach  
✅ **Future-Proof**: Won't break when Roblox updates endpoints  
✅ **More Reliable**: Better rate limiting and error handling  
✅ **Programmatic Access**: Designed for automation  

## Setup Instructions

### Step 1: Create an API Key

1. Go to [Roblox Creator Dashboard](https://create.roblox.com/dashboard/)
2. Navigate to **Credentials** → **API Keys** tab
3. Click **Create API Key**
4. Select **Assets** in the "API System" dropdown
5. Choose your scope:
   - **For personal uploads**: Select your user account
   - **For group uploads**: Select the group
6. Grant **Write** permission
7. Create and copy the key

### Step 2: Configure Asset Reuploader

The application will prompt for your API key on first run. Save it in `api_key.txt` or provide it when prompted.

### Step 3: Run Asset Reuploader

```bash
./AssetReuploader
```

The application will now use the Open Cloud API for uploads.

## Supported Asset Types

- ✅ **Animations** (.rbxm/.rbxmx)
- ✅ **Meshes** (x-file-mesh-data format)
- ✅ **Audio** (MP3, OGG) - Pending API expansion
- ⏳ **Models** (Coming soon)

## API Key Security

⚠️ **IMPORTANT**: Never share your API key publicly!

- Store `api_key.txt` in a safe location
- Don't commit it to version control
- Add to `.gitignore`:
  ```
  api_key.txt
  cookie.txt
  ```

## Troubleshooting

### "403 Forbidden" Error
- Check that your API key has **Write** permission
- Verify the scope includes your user/group
- Make sure you're uploading to the correct account/group

### "410 Gone" Error
- You're still using legacy endpoints
- Update to the latest version of Asset Reuploader
- Ensure API key is properly configured

### Rate Limiting (429 Too Many Requests)
- Open Cloud has rate limits (typically ~100 requests/minute)
- Asset Reuploader implements automatic backoff
- Reduce concurrent uploads if needed

## File Structure

```
internal/roblox/opencloud/
├── client.go         # Main API client
└── upload.go         # Upload handlers

internal/roblox/
└── authenticate_opencloud.go  # OpenCloud auth
```

## Migration Checklist

- [ ] Create Open Cloud API key
- [ ] Update Asset Reuploader to latest version
- [ ] Test upload with a single asset
- [ ] Verify in Roblox Creator Dashboard
- [ ] Scale to batch uploads

## Additional Resources

- [Roblox Open Cloud Documentation](https://create.roblox.com/docs/cloud/reference/apis/assets-api)
- [API Key Management](https://create.roblox.com/dashboard/credentials?activeTab=ApiKeysTab)
- [Asset Types & Formats](https://create.roblox.com/docs/cloud/guides/usage-assets)

## Support

For issues or questions:
1. Check the [Issues](https://github.com/kartFr/Asset-Reuploader/issues) section
2. Join the [Discord community](https://discord.gg/XTEtUqPTat)
3. Review this guide and official Roblox docs
