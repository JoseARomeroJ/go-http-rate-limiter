package limiter

import "errors"

var (
	ErrInvalidRedisClient = errors.New("invalid redis client")
	ErrLimitExceeded      = errors.New("limit exceeded")
	ErrRedis              = errors.New("redis error")
)
