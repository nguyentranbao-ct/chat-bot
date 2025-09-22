package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name       string             `bson:"name" json:"name"`
	Email      string             `bson:"email" json:"email" validate:"required,email"`
	ChatMode   string             `bson:"chat_mode" json:"chat_mode"`
	IsActive   bool               `bson:"is_active" json:"is_active"`
	IsInternal bool               `bson:"is_internal" json:"is_internal"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

type UserAttribute struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"`
	Key       string             `bson:"key" json:"key" validate:"required"`
	Value     string             `bson:"value" json:"value" validate:"required"`
	Tags      []string           `bson:"tags" json:"tags"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
