package handlers

import (
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	Users     *mongo.Collection
	Polls     *mongo.Collection
	Redis     *redis.Client
	JWTSecret []byte
}

func New(database *mongo.Database, rdb *redis.Client, secret string) *Handler {
	return &Handler{
		Users:     database.Collection("users"),
		Polls:     database.Collection("polls"),
		Redis:     rdb,
		JWTSecret: []byte(secret),
	}
}
