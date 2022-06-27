package limiter

import "errors"

var (
	ErrLimitExceeded = errors.New("limit exceeded")
)
