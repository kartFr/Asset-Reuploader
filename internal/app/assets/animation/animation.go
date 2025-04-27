package animation

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartFr/Asset-Reuploader/internal/app/assets/shared/assetutils"
	"github.com/kartFr/Asset-Reuploader/internal/app/assets/shared/clientutils"
	"github.com/kartFr/Asset-Reuploader/internal/app/assets/shared/uploaderror"
	"github.com/kartFr/Asset-Reuploader/internal/app/context"
	"github.com/kartFr/Asset-Reuploader/internal/app/request"
	"github.com/kartFr/Asset-Reuploader/internal/app/response"
	"github.com/kartFr/Asset-Reuploader/internal/retry"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/assetdelivery"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/develop"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/ide"
	"github.com/kartFr/Asset-Reuploader/internal/taskqueue"
)

const assetTypeID int32 = 24

var ErrUnauthorized = errors.New("authentication required to access asset")

func Reupload(ctx *context.Context, r *request.Request) {
	client := ctx.Client
	logger := ctx.Logger
	pauseController := ctx.PauseController
	resp := ctx.Response

	idsToUpload := len(r.IDs)
	var idsProcessed atomic.Int32

	var groupID int64
	if r.IsGroup {
		groupID = r.CreatorID
	}

	filter := assetutils.NewFilter(ctx, r, assetTypeID)
	uploadQueue := taskqueue.New[int64](time.Minute, 3000)

	logger.Println("Reuploading animations...")

	newBatchError := func(amt int, m string, err any) {
		end := int(idsProcessed.Add(int32(amt)))
		start := end - amt
		logger.Error(uploaderror.NewBatch(start, end, idsToUpload, m, err))
	}

	newUploadError := func(m string, assetInfo *develop.AssetInfo, err any) {
		newValue := idsProcessed.Add(1)
		logger.Error(uploaderror.New(int(newValue), idsToUpload, m, assetInfo, err))
	}

	uploadAsset := func(wg *sync.WaitGroup, assetInfo *develop.AssetInfo, location string) {
		defer wg.Done()

		oldName := assetInfo.Name

		assetData, err := clientutils.GetRequest(client, location)
		if err != nil {
			newUploadError("Failed to get asset data", assetInfo, err)
			return
		}

		uploadHandler, err := ide.NewUploadAnimationHandler(client, assetInfo.Name, "", assetData, groupID)
		if err != nil {
			newUploadError("Failed to get upload handler", assetInfo, err)
			return
		}

		res := <-uploadQueue.QueueTask(func() (int64, error) {
			return retry.Do(
				retry.NewOptions(retry.Tries(3)),
				func(try int) (int64, error) {
					pauseController.WaitIfPaused()
					if try > 1 {
						uploadQueue.Limiter.Wait()
					}

					id, err := uploadHandler()
					if err == nil {
						return id, nil
					}

					if err == ide.UploadAnimationErrors.ErrNotLoggedIn {
						clientutils.GetNewCookie(ctx, r, "cookie expired")
					} else if err == ide.UploadAnimationErrors.ErrInappropriateName {
						assetInfo.Name = fmt.Sprintf("(%s) [Censored]", assetInfo.Name)
					}

					return 0, &retry.ContinueRetry{Err: err}
				},
			)
		})

		if err := res.Error; err != nil {
			assetInfo.Name = oldName
			newUploadError("Failed to upload", assetInfo, err)
			return
		}

		newID := res.Result
		newValue := idsProcessed.Add(1)
		logger.Success(uploaderror.New(int(newValue), idsToUpload, "", assetInfo, newID))
		resp.AddItem(response.ResponseItem{
			OldID: assetInfo.ID,
			NewID: newID,
		})
	}

	batchProcess := func(wg *sync.WaitGroup, res assetutils.AssetsInfoResult, batchSize int) {
		defer wg.Done()
		assetsInfo := res.Result

		if err := res.Error; err != nil {
			newBatchError(batchSize, "Failed to get assets info", err)
			return
		}

		filteredInfo := filter(assetsInfo)
		filteredInfoLength := len(filteredInfo)
		idsProcessed.Add(int32(batchSize - filteredInfoLength))
		if len(filteredInfo) == 0 {
			return
		}

		ids := make([]int64, filteredInfoLength)
		for i, assetInfo := range filteredInfo {
			ids[i] = assetInfo.ID
		}
		body := assetutils.NewBatchBodyFromIDs(ids)

		handler, err := assetdelivery.NewBatchHandler(client, body)
		if err != nil {
			newBatchError(filteredInfoLength, "Failed to get batch asset delivery handler", err)
			return
		}

		assetLocations, err := retry.Do(
			retry.NewOptions(retry.Tries(3)),
			func(_ int) ([]*assetdelivery.AssetLocation, error) {
				pauseController.WaitIfPaused()

				locations, err := handler()
				if err != nil {
					return locations, &retry.ContinueRetry{Err: err}
				}

				for _, assetLocation := range locations {
					errs := assetLocation.Errors
					if errs == nil {
						continue
					}
					if errs[0].Message == "Authentication required to access Asset." {
						clientutils.GetNewCookie(ctx, r, "cookie expired")
						return locations, &retry.ContinueRetry{Err: ErrUnauthorized}
					}
				}

				return locations, nil
			},
		)
		if err != nil {
			newBatchError(filteredInfoLength, "Failed to get asset locations", err)
			return
		}

		var uploadWG sync.WaitGroup
		uploadWG.Add(filteredInfoLength)
		for i, assetInfo := range filteredInfo {
			locationInfo := assetLocations[i]

			if errors := locationInfo.Errors; errors != nil {
				newUploadError("Failed to get asset location for", assetInfo, errors[0].Message)
				uploadWG.Done()
				continue
			}

			location := locationInfo.Locations[0].Location
			go uploadAsset(&uploadWG, assetInfo, location)
		}
		uploadWG.Wait()
	}

	var wg sync.WaitGroup
	tasks := assetutils.GetAssetsInfoInChunks(ctx, r)
	wg.Add(len(tasks))
	for i, task := range tasks {
		batchSize := 50
		if i == len(tasks)-1 {
			batchSize = idsToUpload % 50
			if batchSize == 0 {
				batchSize = 50
			}
		}

		go batchProcess(&wg, <-task, batchSize)
	}
	wg.Wait()
}
