package limiter

import "net/http"

func (l *redisLimiter) LimitHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := l.CheckLimitFromRequest(r)

		if err != nil && err == ErrLimitExceeded {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		} else if err != nil && err == ErrRedis {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
