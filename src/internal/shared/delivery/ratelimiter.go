package delivery

import (
	"github.com/Aboody-Studios/ballr/src/internal/shared/infrastructure"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(rdb *redis.Client) echo.MiddlewareFunc {
	return echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		IdentifierExtractor: ExtractEmailFromJWT,
		Store:               &infrastructure.RedisStore{Client: rdb},
	})

}
