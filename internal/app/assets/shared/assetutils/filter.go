package assetutils

import (
	"github.com/kartFr/Asset-Reuploader/internal/app/context"
	"github.com/kartFr/Asset-Reuploader/internal/app/request"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/develop"
)

func NewFilter(ctx *context.Context, r *request.Request, assetTypeID int32) func(assetsInfo develop.GetAssetsInfoResponse) []*develop.AssetInfo {
	creatorID := r.CreatorID
	userID := ctx.Client.UserInfo.ID
	checkUserID := !r.IsGroup

	return func(assetsInfo develop.GetAssetsInfoResponse) []*develop.AssetInfo {
		filteredAssetsInfo := assetsInfo.Data[:0]
		for _, info := range assetsInfo.Data {
			if info.TypeID != assetTypeID {
				continue
			}

			assetCreatorID := info.Creator.TargetID
			if assetCreatorID == creatorID || assetCreatorID == 1 || (checkUserID && assetCreatorID == userID) {
				continue
			}

			filteredAssetsInfo = append(filteredAssetsInfo, info)
		}
		return filteredAssetsInfo
	}
}
