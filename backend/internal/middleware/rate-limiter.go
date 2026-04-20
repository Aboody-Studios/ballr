package middleware

import (
	"github.com/Aboody-Studios/ballr/backend/internal/handlers"
	"github.com/Aboody-Studios/ballr/backend/internal/infrastructure"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(rdb *redis.Client) echo.MiddlewareFunc {
	return echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		IdentifierExtractor: handlers.ExtractEmailFromJWT,
		Store:               &infrastructure.RedisStore{Client: rdb},
	})

}
