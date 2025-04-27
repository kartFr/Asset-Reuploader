package assetutils

import (
	"net"
	"time"

	"github.com/kartFr/Asset-Reuploader/internal/app/assets/shared/clientutils"
	"github.com/kartFr/Asset-Reuploader/internal/app/context"
	"github.com/kartFr/Asset-Reuploader/internal/app/request"
	"github.com/kartFr/Asset-Reuploader/internal/retry"
	"github.com/kartFr/Asset-Reuploader/internal/roblox/develop"
	"github.com/kartFr/Asset-Reuploader/internal/taskqueue"
)

const (
	assetsInfoRateLimit time.Duration = time.Minute / 1
	AssetsInfoChunkSize int           = 50
)

type AssetsInfoResult = taskqueue.TaskResult[develop.GetAssetsInfoResponse]

func GetAssetsInfoInChunks(ctx *context.Context, r *request.Request) []chan AssetsInfoResult {
	queue := taskqueue.New[develop.GetAssetsInfoResponse](time.Minute, 100)

	newAssetsInfoHandler := func(ids []int64) func() (develop.GetAssetsInfoResponse, error) {
		return func() (develop.GetAssetsInfoResponse, error) {
			handler, err := develop.NewAssetsInfoHandler(ctx.Client, ids)
			if err != nil {
				return develop.GetAssetsInfoResponse{}, err
			}

			return retry.Do(
				retry.NewOptions(retry.Tries(3)),
				func(try int) (develop.GetAssetsInfoResponse, error) {
					ctx.PauseController.WaitIfPaused()
					if try > 1 {
						queue.Limiter.Wait()
					}

					assetsInfo, err := handler()
					if err == nil {
						return assetsInfo, nil
					}

					if err == develop.GetAssetsInfoErrors.ErrUnauthorized {
						clientutils.GetNewCookie(ctx, r, "cookie expired")
					}

					switch err.(type) {
					case *net.OpError, *net.DNSError:
						queue.Limiter.Decrement()
					}

					return develop.GetAssetsInfoResponse{}, &retry.ContinueRetry{Err: err}
				},
			)
		}
	}

	ids := r.IDs

	chunkAmount := (len(ids) + AssetsInfoChunkSize - 1) / AssetsInfoChunkSize
	tasks := make([]chan AssetsInfoResult, 0, chunkAmount)
	for start, end := 0, 50; start < len(ids); start, end = start+50, end+50 {
		end = min(end, len(ids))
		idChunk := ids[start:end]
		tasks = append(tasks, queue.QueueTask(newAssetsInfoHandler(idChunk)))
	}

	return tasks
}
