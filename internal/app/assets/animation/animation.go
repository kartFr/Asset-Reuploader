package animation

import (
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
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
	"github.com/kartFr/Asset-Reuploader/internal/roblox/assetdelivery"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/develop"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/games"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/opencloud"
	"github.com/kartFr/Asset-Reuploader/internal/shardedmap"
	"github.com/kartFr/Asset-Reuploader/internal/taskqueue"
)

const assetTypeID int32 = 24

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

const maxRetryDepth = 3

func Reupload(ctx *context.Context, r *request.Request) {
	reuploadWithRetry(ctx, r, 0)
}

func reuploadWithRetry(ctx *context.Context, r *request.Request, retryDepth int) {
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

	filter := assetutils.NewFilter(ctx, r, assetTypeID)

	creatorPlaceMap := shardedmap.New[*atomicarray.AtomicArray[int64]]()
	creatorMutexMap := shardedmap.New[*sync.RWMutex]()

	uploadWindow, uploadLimit := config.GetUploadQueueConfig()
	groupGameWindow, groupGameLimit := config.GetGroupGameQueueConfig()
	userGameWindow, userGameLimit := config.GetUserGameQueueConfig()

	uploadQueue := taskqueue.New[int64](uploadWindow, uploadLimit)
	groupGameQueue := taskqueue.New[*games.GamesResponse](groupGameWindow, groupGameLimit)
	userGameQueue := taskqueue.New[*games.GamesResponse](userGameWindow, userGameLimit)
	batchSemaphore := make(chan struct{}, 5)

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

		uploadHandler, err := opencloud.NewUploadAnimationHandler(client, assetInfo.Name, "", assetData, groupID)
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

					switch err {
					case opencloud.UploadAnimationErrors.ErrNotAuthenticated, opencloud.UploadAnimationErrors.ErrInvalidAPIKey:
						clientutils.GetNewAPIKey(ctx, "api key invalid or expired")
					case opencloud.UploadAnimationErrors.ErrRateLimited:
						time.Sleep(time.Duration(try+1) * time.Second)
						uploadQueue.Limiter.Decrement()
					case opencloud.UploadAnimationErrors.ErrInappropriateName:
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
			queueRes := <-groupGameQueue.QueueTask(func() (*games.GamesResponse, error) {
				return games.GroupGames(client, creatorID)
			})
			resp = queueRes.Result
			err = queueRes.Error
		} else {
			queueRes := <-userGameQueue.QueueTask(func() (*games.GamesResponse, error) {
				return games.UserGames(client, creatorID)
			})
			resp = queueRes.Result
			err = queueRes.Error
		}
		if err != nil {
			return nil, err
		}

		ids := make([]int64, 0, len(defaultPlaceIDs)) // we only do len defaultPlaceIds because there may be overlapping, i guess allocating more memory would be fine... idk guys im getting lazy just wait for revamp
		for _, placeInfo := range resp.Data {         // yes we copying many bytes per iteration, yes i dont care, yes this is another stupid message, yes code iwll get better on revamp :sob:
			rootPlaceID := placeInfo.RootPlace.ID

			if _, exists := defaultPlaceIDsMap[rootPlaceID]; exists {
				continue
			}
			ids = append(ids, rootPlaceID) // we no longer only need 1 valid place id :// ( ͡° ͜ʖ ͡°) yall remember this peak face lmk
		}
		ids = append(ids, defaultPlaceIDs...)

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
		creatorPlaceCache := placeCache.Load()
		for _, placeID := range creatorPlaceCache {
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
			if hadSuccess && len(creatorPlaceCache) > 1 {
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
		if filteredInfoLength == 0 {
			return
		}

		CreatorAssets := make(map[string]map[int64][]*develop.AssetInfo)
		for _, assetInfo := range filteredInfo {
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

	var failedIDs []int64
	var failedIDsMutex sync.Mutex

	addFailedID := func(id int64) {
		failedIDsMutex.Lock()
		failedIDs = append(failedIDs, id)
		failedIDsMutex.Unlock()
	}

	origNewUploadError := newUploadError
	newUploadError = func(m string, assetInfo *develop.AssetInfo, err any) {
		addFailedID(assetInfo.ID)
		origNewUploadError(m, assetInfo, err)
	}

	origNewBatchError := newBatchError
	newBatchError = func(amt int, m string, err any) {
		origNewBatchError(amt, m, err)
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

	if len(failedIDs) == 0 {
		return
	}

	if retryDepth >= maxRetryDepth {
		color.Error.Println(fmt.Sprintf("%d animation(s) still failed after %d retries. Giving up.", len(failedIDs), maxRetryDepth))
		os.Remove("failed_animations.txt")
		return
	}

	color.Warn.Println(fmt.Sprintf("%d animation(s) failed, retrying (attempt %d/%d)...", len(failedIDs), retryDepth+1, maxRetryDepth))

	failedIDsFile := "failed_animations.txt"
	strIDs := make([]string, len(failedIDs))
	for i, id := range failedIDs {
		strIDs[i] = strconv.FormatInt(id, 10)
	}
	os.WriteFile(failedIDsFile, []byte(strings.Join(strIDs, "\n")), 0644)

	retryRequest := &request.Request{
		UniverseID:      r.UniverseID,
		PlaceID:         r.PlaceID,
		CreatorID:       r.CreatorID,
		IDs:             failedIDs,
		DefaultPlaceIDs: r.DefaultPlaceIDs,
		IsGroup:         r.IsGroup,
	}

	reuploadWithRetry(ctx, retryRequest, retryDepth+1)

	os.Remove(failedIDsFile)
	color.Success.Println("Failed animation retry complete.")
}
