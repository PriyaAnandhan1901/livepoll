package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const UserIDKey = "userID"

// Auth verifies the Bearer JWT and stores the user id in the context.
func Auth(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "login required"})
			return
		}
		claims := &jwt.RegisteredClaims{}
		tok, err := jwt.ParseWithClaims(parts[1], claims,
			func(t *jwt.Token) (any, error) { return secret, nil },
			jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !tok.Valid || claims.Subject == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(UserIDKey, claims.Subject)
		c.Next()
	}
}

// RateLimit is a Redis-backed fixed-window limiter keyed by client IP.
// It fails open if Redis is unavailable.
func RateLimit(rdb *redis.Client, name string, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", name, c.ClientIP())
		ctx := c.Request.Context()
		n, err := rdb.Incr(ctx, key).Result()
		if err == nil {
			if n == 1 {
				rdb.Expire(ctx, key, window)
			}
			if n > limit {
				c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
				return
			}
		}
		c.Next()
	}
}

// MaxBody caps request body size.
func MaxBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}
