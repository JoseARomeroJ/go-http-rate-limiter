package limiter

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"time"
)

type redisLimiter struct {
	client *redis.Client
}

func CreateRedisRateLimiter(ctx context.Context, name string, r *redis.Client, configurations map[uint32]GeneralLimitConfiguration,
	getKeyTypeFunc func(r *http.Request) (string, uint32)) Limiter {

	if r == nil {
		panic("invalid redis client")
	}

	var l = limiter{
		name:                  name,
		context:               ctx,
		getKeyFromRequestFunc: getKeyTypeFunc,
		storageKeyGen:         defaultStorageKeyGen,
		limitCacheHandler: &redisLimiter{
			client: r,
		},
		configurations: configurations,
	}

	return &l
}

func (l *redisLimiter) clearExpiredRequests(ctx context.Context, key string, duration time.Duration) error {
	min := time.Now().Add(-duration).UnixMilli()

	removeByScore := l.client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(min, 10))
	return removeByScore.Err()
}

func (l *redisLimiter) addNewRequest(ctx context.Context, key string) error {
	rid := uuid.New()
	now := time.Now().UnixMilli()

	add := l.client.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now),
		Member: rid.String(),
	})

	return add.Err()
}

func (l *redisLimiter) getRequestCount(ctx context.Context, key string) (uint64, error) {
	count := l.client.ZCount(ctx, key, "-inf", "+inf")
	c, err := count.Result()

	return uint64(c), err
}
