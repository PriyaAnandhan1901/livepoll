package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"livepoll/config"
	"livepoll/db"
	"livepoll/handlers"
	"livepoll/middleware"
)

func main() {
	_ = godotenv.Load() // optional .env for local dev
	cfg := config.Load()

	ctx := context.Background()
	database, err := db.ConnectMongo(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	rdb, err := db.ConnectRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	h := handlers.New(database, rdb, cfg.JWTSecret)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.MaxBody(1<<20))
	r.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(cfg.CORSOrigin, ","),
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:       12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		if rdb.Ping(c.Request.Context()).Err() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "redis down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	authLimit := middleware.RateLimit(rdb, "auth", 10, time.Minute)
	api.POST("/auth/signup", authLimit, h.Signup)
	api.POST("/auth/login", authLimit, h.Login)

	// public: anyone with the link can view and vote
	api.GET("/polls/:id", h.GetPoll)
	api.GET("/polls/:id/stream", h.Stream)
	api.POST("/polls/:id/vote", middleware.RateLimit(rdb, "vote", 30, time.Minute), h.Vote)

	// protected: create / manage
	authed := api.Group("", middleware.Auth([]byte(cfg.JWTSecret)))
	authed.POST("/polls", h.CreatePoll)
	authed.GET("/my/polls", h.MyPolls)
	authed.POST("/polls/:id/close", h.ClosePoll)
	authed.DELETE("/polls/:id", h.DeletePoll)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		// no WriteTimeout: SSE connections are long-lived
	}
	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
