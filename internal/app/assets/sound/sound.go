package sound

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kartFr/Asset-Reuploader/internal/app/assets/shared/assetutils"
	"github.com/kartFr/Asset-Reuploader/internal/app/assets/shared/clientutils"
	"github.com/kartFr/Asset-Reuploader/internal/app/assets/shared/uploaderror"
	"github.com/kartFr/Asset-Reuploader/internal/app/config"
	"github.com/kartFr/Asset-Reuploader/internal/app/context"
	"github.com/kartFr/Asset-Reuploader/internal/app/request"
	"github.com/kartFr/Asset-Reuploader/internal/app/response"
	"github.com/kartFr/Asset-Reuploader/internal/atomicarray"
	"github.com/kartFr/Asset-Reuploader/internal/color"
	"github.com/kartFr/Asset-Reuploader/internal/retry"
	"github.com/kartFr/Asset-Reuploader/internal/roblox"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/assetdelivery"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/assets"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/develop"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/games"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/publish"
	"github.com/kartFr/Asset-Reuploader/internal/shardedmap"
	"github.com/kartFr/Asset-Reuploader/internal/taskqueue"
)

const assetTypeID int32 = 3

var ErrUnauthorized = errors.New("authentication required to access asset")

func MoveValueToTop[T comparable](arr *atomicarray.AtomicArray[T], value T) {
	arr.Update(func(currentArray []T) []T {
		if currentArray[0] == value {
			return nil
		}

		for i, v := range currentArray {
			if v != value {
				continue
			}
			if i == 1 {
				currentArray[0], currentArray[1] = currentArray[1], currentArray[0]
				return currentArray
			}

			copy(currentArray[1:i+1], currentArray[0:i])
			currentArray[0] = value
			return currentArray
		}

		return nil
	})
}

func Reupload(ctx *context.Context, r *request.Request) {
	client := ctx.Client
	logger := ctx.Logger
	pauseController := ctx.PauseController
	resp := ctx.Response

	idsToUpload := len(r.IDs)
	var idsProcessed atomic.Int32

	defaultPlaceIDs := r.DefaultPlaceIDs
	defaultPlaceIDsMap := make(map[int64]struct{}, len(defaultPlaceIDs))
	for _, placeID := range defaultPlaceIDs {
		defaultPlaceIDsMap[placeID] = struct{}{}
	}

	var groupID int64
	if r.IsGroup {
		groupID = r.CreatorID
	}
	currentPlaceID := r.PlaceID

	filter := assetutils.NewFilter(ctx, r, assetTypeID)

	creatorPlaceMap := shardedmap.New[*atomicarray.AtomicArray[int64]]()
	creatorMutexMap := shardedmap.New[*sync.RWMutex]()

	uploadWindow, uploadLimit := config.GetUploadQueueConfig()
	uploadQueue := taskqueue.New[int64](uploadWindow, uploadLimit)
	permissionQueue := taskqueue.New[*assets.PermissionResponse](time.Minute, 60)
	permissionRequest := assetutils.NewPermissionBodyFromIds([]int64{r.UniverseID})
	batchSemaphore := make(chan struct{}, 5)

	logger.Println("Reuploading sounds...")

	newBatchError := func(amt int, m string, err any) {
		end := int(idsProcessed.Add(int32(amt)))
		start := end - amt
		logger.Error(uploaderror.NewBatch(start, end, idsToUpload, m, err))
	}

	newUploadError := func(m string, assetInfo *develop.AssetInfo, err any) {
		newValue := idsProcessed.Add(1)
		logger.Error(uploaderror.New(int(newValue), idsToUpload, m, assetInfo, err))
	}

	grantPermissions := func(newID int64) (*assets.PermissionResponse, error) {
		permissionsClient, _ := roblox.NewClient("")
		permissionsClient.Cookie = client.Cookie
		permissionsClient.SetToken(client.GetToken())

		permissionHandler, err := assets.NewUpdatePermissionsHandler(client, newID, permissionRequest)
		if err != nil {
			return nil, err
		}

		res := <-permissionQueue.QueueTask(func() (*assets.PermissionResponse, error) {
			return retry.Do(
				retry.NewOptions(retry.Tries(3)),
				func(try int) (*assets.PermissionResponse, error) {
					pauseController.WaitIfPaused()
					if try > 1 {
						uploadQueue.Limiter.Wait()
					}

					permissionReponse, err := permissionHandler()
					if err == nil {
						return permissionReponse, nil
					}

					if err == assets.UpdatePermissionErrors.ErrNotAuthenticated {
						return nil, &retry.ExitRetry{Err: err}
					} else {
						switch err.(type) {
						case *net.OpError, *net.DNSError:
							permissionQueue.Limiter.Decrement()
						}
					}

					return nil, &retry.ContinueRetry{Err: err}
				},
			)
		})
		return res.Result, res.Error
	}

	uploadAsset := func(wg *sync.WaitGroup, assetInfo *develop.AssetInfo, location string) {
		defer wg.Done()

		oldName := assetInfo.Name

		assetData, err := clientutils.GetRequest(client, location)
		if err != nil {
			newUploadError("Failed to get asset data", assetInfo, err)
			return
		}

		uploadHandler, err := publish.NewUploadAudioHandler(client, assetInfo.Name, assetData, groupID)
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

					uploadResponse, err := uploadHandler()
					if err == nil {
						return uploadResponse.ID, nil
					}

					switch err {
					case publish.UploadAudioErrors.ErrNotAuthenticated:
						clientutils.GetNewCookie(ctx, r, "cookie expired")
					case publish.UploadAudioErrors.ErrQuotaExceeded:
						clientutils.GetNewCookie(ctx, r, "audio limit exceeded")
					case publish.UploadAudioErrors.ErrModerated:
						assetInfo.Name = fmt.Sprintf("(%s) [Censored]", assetInfo.Name)
					default:
						switch err.(type) {
						case *net.OpError, *net.DNSError:
							uploadQueue.Limiter.Decrement()
						}
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

		if _, err = grantPermissions(newID); err != nil {
			message := fmt.Sprintf(">> %s(%d) failed to grant permission: ", assetInfo.Name, newID) + err.Error()
			if pauseController.IsPaused {
				color.Error.Fprintln(logger.History, message)
			} else {
				logger.Error(message)
			}
		} else {
			message := fmt.Sprintf(">> %s(%d) granted permission", assetInfo.Name, newID)
			if pauseController.IsPaused {
				color.Info.Fprintln(logger.History, message)
			} else {
				logger.Info(message)
			}
		}
	}

	getCreatorPlaceCache := func(creatorID int64, creatorType string) (*atomicarray.AtomicArray[int64], error) {
		creatorShard, exists := creatorPlaceMap.GetShard(creatorType)
		mutexShard, _ := creatorMutexMap.GetShard(creatorType)
		if !exists {
			creatorShard = creatorPlaceMap.NewShard(creatorType)
			mutexShard = creatorMutexMap.NewShard(creatorType)
		}

		if cache, cacheExists := creatorShard.Get(creatorID); cacheExists {
			return cache, nil
		}

		mutex, mutexExists := mutexShard.Get(creatorID)
		if !mutexExists {
			mutex = &sync.RWMutex{}
			mutexShard.Set(creatorID, mutex)
		}

		mutex.Lock()
		defer mutex.Unlock()

		if cache, cacheExists := creatorShard.Get(creatorID); cacheExists {
			return cache, nil
		}

		var resp *games.GamesResponse
		var err error
		if creatorType == "Group" {
			resp, err = games.GroupGames(client, creatorID)
		} else {
			resp, err = games.UserGames(client, creatorID)
		}
		if err != nil {
			return nil, err
		}

		ids := make([]int64, 0, len(defaultPlaceIDs)+len(resp.Data))
		ids = append(ids, defaultPlaceIDs...)
		for _, placeInfo := range resp.Data {
			rootPlaceID := placeInfo.RootPlace.ID

			if _, exists := defaultPlaceIDsMap[rootPlaceID]; exists {
				continue
			}

			ids = append(ids, rootPlaceID)
		}

		cache := atomicarray.New(&ids)
		creatorShard.Set(creatorID, cache)
		mutexShard.Remove(creatorID)
		return cache, nil
	}

	getAssetLocations := func(body []*assetdelivery.AssetRequestItem, placeID int64) ([]*assetdelivery.AssetLocation, error) {
		handler, err := assetdelivery.NewBatchHandler(client, body, placeID)
		if err != nil {
			return nil, err
		}

		return retry.Do(
			retry.NewOptions(retry.Tries(3)),
			func(try int) ([]*assetdelivery.AssetLocation, error) {
				pauseController.WaitIfPaused()

				locations, err := handler()
				if err != nil {
					if err == assetdelivery.ErrRateLimited {
						time.Sleep(time.Duration(try+1) * time.Second)
					}
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
	}

	batchUpload := func(wg *sync.WaitGroup, creatorID int64, creatorType string, creatorAssets []*develop.AssetInfo) {
		defer wg.Done()

		placeCache, err := getCreatorPlaceCache(creatorID, creatorType)
		if err != nil {
			newBatchError(len(creatorAssets), "Failed to get creator places", err)
		}

		assetInfoMap := make(map[int64]*develop.AssetInfo)
		ids := make([]int64, len(creatorAssets))
		for i, assetInfo := range creatorAssets {
			ids[i] = assetInfo.ID
			assetInfoMap[assetInfo.ID] = assetInfo
		}
		body := assetutils.NewBatchBodyFromIDs(ids)

		var uploadWG sync.WaitGroup
		var assetLocations []*assetdelivery.AssetLocation
		for _, placeID := range placeCache.Load() {
			assetLocations, err = getAssetLocations(body, placeID)
			if err != nil {
				newBatchError(len(body), "Failed to get asset locations", err)
				return
			}

			var hadSuccess bool
			for assetIndex, assetLocation := range slices.Backward(assetLocations) {
				if len(assetLocation.Locations) == 0 {
					continue
				}
				hadSuccess = true

				assetID := body[assetIndex].AssetID
				body = slices.Delete(body, assetIndex, assetIndex+1)

				uploadWG.Add(1)
				go uploadAsset(&uploadWG, assetInfoMap[assetID], assetLocation.Locations[0].Location)
			}
			if hadSuccess {
				MoveValueToTop(placeCache, placeID)
			}
			if len(body) == 0 {
				break
			}
		}

		var index int
		for _, assetLocation := range assetLocations {
			if len(assetLocation.Locations) != 0 {
				continue
			}
			assetID := body[index].AssetID
			index++

			assetInfo := assetInfoMap[assetID]
			newUploadError("Failed to get asset location", assetInfo, assetLocation.Errors[0].Message)
		}

		uploadWG.Wait()
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

		assetLocations, err := getAssetLocations(body, currentPlaceID)
		if err != nil {
			newBatchError(filteredInfoLength, "Failed to get asset locations to see permissions", err)
		}

		unownedAssets := make([]*develop.AssetInfo, 0)
		for i, location := range assetLocations {
			if len(location.Locations) > 0 {
				continue
			}

			unownedAssets = append(unownedAssets, filteredInfo[i])
		}

		CreatorAssets := make(map[string]map[int64][]*develop.AssetInfo)
		for _, assetInfo := range unownedAssets {
			assetCreatorType := assetInfo.Creator.Type
			assetCreatorID := assetInfo.Creator.TargetID

			creatorType, exists := CreatorAssets[assetCreatorType]
			if !exists {
				creatorType = make(map[int64][]*develop.AssetInfo)
				CreatorAssets[assetCreatorType] = creatorType
			}

			creatorAssets, exists := creatorType[assetCreatorID]
			if !exists {
				creatorAssets = make([]*develop.AssetInfo, 0)
				creatorType[assetCreatorID] = creatorAssets
			}

			creatorType[assetCreatorID] = append(creatorAssets, assetInfo)
		}

		var uploadWG sync.WaitGroup
		for creatorType, creatorAssetMap := range CreatorAssets {
			uploadWG.Add(len(creatorAssetMap))

			for creatorID, creatorAssets := range creatorAssetMap {
				batchSemaphore <- struct{}{}
				go func(creatorID int64, creatorType string, creatorAssets []*develop.AssetInfo) {
					defer func() { <-batchSemaphore }()
					batchUpload(&uploadWG, creatorID, creatorType, creatorAssets)
				}(creatorID, creatorType, creatorAssets)
			}
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
