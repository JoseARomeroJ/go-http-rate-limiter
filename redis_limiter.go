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

	key = key + l.name

	c, ok := l.configurations[t]
	if !ok {
		return ErrLimitExceeded
	}

	p := l.client.Pipeline()
	ctx := r.Context()

	expiredResult := l.clearExpiredRequests(ctx, p, key, c.Duration)
	addResult := l.addNewRequest(ctx, p, key)
	countResult := l.getRequestCount(ctx, p, key)

	count, countErr := countResult.Result()

	if _, err := p.Exec(ctx); err != nil {
		return ErrRedis
	} else if expiredResult.Err() != nil || addResult.Err() != nil || countErr != nil {
		return ErrRedis
	}

	if uint64(count) > uint64(c.RequestLimit) {
		return ErrLimitExceeded
	}

	return nil
}

func (l *redisLimiter) clearExpiredRequests(ctx context.Context, p redis.Pipeliner, key string, duration time.Duration) *redis.IntCmd {
	min := time.Now().Add(-duration)

	removeByScore := p.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(min.UnixMilli(), 10))
	return removeByScore
}

func (l *redisLimiter) addNewRequest(ctx context.Context, p redis.Pipeliner, key string) *redis.IntCmd {
	rid := uuid.New()

	add := p.ZAdd(ctx, key, &redis.Z{
		Score:  float64(time.Now().UnixMilli()),
		Member: rid,
	})

	return add
}

func (l *redisLimiter) getRequestCount(ctx context.Context, p redis.Pipeliner, key string) *redis.IntCmd {
	count := p.ZCount(ctx, key, "-inf", "+inf")
	return count
}
