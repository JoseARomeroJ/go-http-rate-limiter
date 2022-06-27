package limiter

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultStorageKeyPreset = "hrl-%s-"
)

var (
	defaultStorageKeyGen = func(key string) string {
		return fmt.Sprintf(defaultStorageKeyPreset, key)
	}
)

type Limiter interface {
	LimitHandler(next http.Handler) http.Handler
	CheckLimitFromRequest(r *http.Request) error
}

type limiter struct {
	name                  string
	context               context.Context
	getKeyFromRequestFunc func(r *http.Request) (string, uint32)
	storageKeyGen         func(key string) string
	limitCacheHandler     limitCacheHandler
	configurations        map[uint32]LimitConfiguration
}

type limitCacheHandler interface {
	clearExpiredRequests(ctx context.Context, key string, duration time.Duration) error
	addNewRequest(ctx context.Context, key string) error
	getRequestCount(ctx context.Context, key string) (uint64, error)
}

type LimitConfiguration struct {
	RequestLimit uint32        `json:"limit"`
	Duration     time.Duration `json:"duration"`
}

func (l limiter) CheckLimitFromRequest(r *http.Request) error {
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

	if expiredErr := l.limitCacheHandler.clearExpiredRequests(ctx, key, c.Duration); expiredErr != nil {
		return expiredErr
	} else if addErr := l.limitCacheHandler.addNewRequest(ctx, key); addErr != nil {
		return addErr
	} else if num, err := l.limitCacheHandler.getRequestCount(ctx, key); err != nil {
		return err
	} else {
		count = num
	}

	if count > uint64(c.RequestLimit) {
		return ErrLimitExceeded
	}

	return nil
}
