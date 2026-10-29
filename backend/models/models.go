package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	CreatedAt    time.Time          `bson:"created_at" json:"createdAt"`
}

type Option struct {
	Text  string `bson:"text" json:"text"`
	Votes int64  `bson:"votes" json:"votes"` // durable backup of the Redis counter
}

type Poll struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Question  string             `bson:"question" json:"question"`
	Options   []Option           `bson:"options" json:"options"`
	OwnerID   primitive.ObjectID `bson:"owner_id" json:"-"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
	ExpiresAt *time.Time         `bson:"expires_at,omitempty" json:"expiresAt,omitempty"`
	Closed    bool               `bson:"closed" json:"closed"`
}
