# Unreleased

- Fixed saving cookie for mac users. ([#27](https://github.com/kartFr/Asset-Reuploader/pull/27))

# [1.3.1](https://github.com/kartFr/Asset-Reuploader/releases/tag/1.3.1) - April 27th, 2025

- Changed client timeout from `15s` to `30s` ([#24](https://github.com/kartFr/Asset-Reuploader/pull/24))

# [1.3.0](https://github.com/kartFr/Asset-Reuploader/releases/tag/1.3.0) - April 27th, 2025

- Added a changelog. Adds changelog to releases ([#16](https://github.com/kartFr/Asset-Reuploader/pull/16))

## Executable

- Added fixed window limiter instead of naively sleeping. (big change very good for reuploading 😇) ([#18](https://github.com/kartFr/Asset-Reuploader/pull/18))
- Fixed `ErrNoCreateItemPermission` saying `permissios` instead of `permission`. ([#19](https://github.com/kartFr/Asset-Reuploader/pull/19))
- Fixed blank error messages ([#21](https://github.com/kartFr/Asset-Reuploader/pull/21))

## Plugin 1.2.1

- Changed audio tip to be more clear ([#10](https://github.com/kartFr/Asset-Reuploader/pull/10))
- Fixed tip bar theme not updating

# [1.2.0](https://github.com/kartFr/Asset-Reuploader/releases/tag/1.2.0) - April 18th, 2025

## Executable

- Changed client context timeout from `10s` to `15s`
- Added ratelimit to reuploading animations. ([#5](https://github.com/kartFr/Asset-Reuploader/pull/5))
- Added micro optimizations that nobody will ever notice.

## Plugin 1.2.0

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

## Plugin 1.1.0

- Changed IDs to replace on new threads.
- Changed Input element to it's own class.
- Added preloading for theme assets.
- Fixed reuploaded selected using filter options. ([#3](https://github.com/kartFr/Asset-Reuploader/pull/3))

# [1.0.0](https://github.com/kartFr/Asset-Reuploader/releases/tag/1.0.0) - April 12th, 2025

Initial release
