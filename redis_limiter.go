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
	limiter
}

func CreateRedisRateLimiter(ctx context.Context, name string, r *redis.Client, configurations map[uint32]LimitConfiguration,
	getKeyTypeFunc func(r *http.Request) (string, uint32)) Limiter {

	if r == nil {
		panic("invalid redis client")
	}

	var l = redisLimiter{
		client: r,
		limiter: limiter{
			name:                  name,
			context:               ctx,
			getKeyFromRequestFunc: getKeyTypeFunc,
			storageKeyGen:         defaultStorageKeyGen,
			configurations:        configurations,
		},
	}
	return &l
}

func (l *redisLimiter) CheckLimitFromRequest(r *http.Request) error {
	if r == nil {
		panic("invalid request")
	}

	key, t := l.getKeyFromRequestFunc(r)
	if key == "" {
		return ErrLimitExceeded
	}

	key = key + "-" + l.name

	c, ok := l.configurations[t]
	if !ok {
		return ErrLimitExceeded
	} else if c.RequestLimit == 0 {
		return nil
	}

	ctx := r.Context()

	var count uint64

	if expiredResult := l.clearExpiredRequests(ctx, key, c.Duration); expiredResult.Err() != nil {
		return ErrRedis
	} else if addResult := l.addNewRequest(ctx, key); addResult.Err() != nil {
		return ErrRedis
	} else if countResult := l.getRequestCount(ctx, key); countResult.Err() != nil {
		return ErrRedis
	} else {
		v, _ := countResult.Result()
		count = uint64(v)
	}

	if uint64(count) > uint64(c.RequestLimit) {
		return ErrLimitExceeded
	}

	return nil
}

func (l *redisLimiter) clearExpiredRequests(ctx context.Context, key string, duration time.Duration) *redis.IntCmd {
	min := time.Now().Add(-duration).UnixMilli()

	removeByScore := l.client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(min, 10))
	return removeByScore
}

func (l *redisLimiter) addNewRequest(ctx context.Context, key string) *redis.IntCmd {
	rid := uuid.New()
	now := time.Now().UnixMilli()

	add := l.client.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now),
		Member: rid.String(),
	})

	return add
}

func (l *redisLimiter) getRequestCount(ctx context.Context, key string) *redis.IntCmd {
	count := l.client.ZCount(ctx, key, "-inf", "+inf")
	return count
}
