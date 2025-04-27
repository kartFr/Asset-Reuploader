# Unreleased

- Added a changelog. Adds changelog to releases ([#7](https://github.com/kartFr/Asset-Reuploader/issues/7))

# [1.2.0](https://github.com/kartFr/Asset-Reuploader/releases/tag/1.2.0) - April 18th, 2025

## Executable

- Changed client context timeout from `10s` to `15s`
- Added ratelimit to reuploading animations. ([#5](https://github.com/kartFr/Asset-Reuploader/pull/5))
- Added micro optimizations that nobody will ever notice.

## Plugin

- Added notification border
- Some other stuff nobody will ever care about (me fr)
- lmk if you read this :p

### Theme changes

- Changed `Foreground` to `MainBackground`
- Added Background image
- Added Text size

# [1.1.0](https://github.com/kartFr/Asset-Reuploader/releases/tag/1.1.0) - April 16th, 2025

## Executable

- Added `Unauthorized` status on sound and mesh reupload requests to provide proper warnings for the plugin.
- Added an editable configuration file.
- Changed client context timeout increased `5s-10s`
- Changed group error messages to be clearer.
- Fixed issue where reuploading wasn't happening in goroutines.

## Plugin

- Changed IDs to replace on new threads.
- Changed Input element to it's own class.
- Added preloading for theme assets.
- Fixed reuploaded selected using filter options. ([#3](https://github.com/kartFr/Asset-Reuploader/pull/3))

# [1.0.0](https://github.com/kartFr/Asset-Reuploader/releases/tag/1.0.0) - April 12th, 2025

Initial release
