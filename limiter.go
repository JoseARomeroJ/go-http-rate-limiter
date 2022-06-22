package limiter

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	defautlStorageKeyPreset = "hrl-%s"
)

var (
	defaultStorageKeyGen = func(key string) string {
		return fmt.Sprintf(defautlStorageKeyPreset, key)
	}
)

type Limiter interface {
	LimitHandler(next http.Handler) http.Handler
	CheckLimitFromRequest(r *http.Request) error
}

type limiter struct {
	context               context.Context
	getKeyFromRequestFunc func(r *http.Request) (string, uint32)
	storageKeyGen         func(key string) string
	configurations        map[uint32]LimitConfiguration
}

type limitCacheHandler interface {
	ClearExpiredRequests(key string, duration time.Duration) error
	AddNewRequest(key string, duration time.Duration) error
	GetRequestCount(key string) (uint64, error)
}

type LimitConfiguration struct {
	RequestLimit uint32        `json:"limit"`
	Duration     time.Duration `json:"duration"`
}
