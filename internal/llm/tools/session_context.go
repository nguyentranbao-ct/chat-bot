package tools

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SessionContext interface {
	GetSessionID() primitive.ObjectID
	GetChannelID() string
	GetUserID() string
	GetSenderID() string

	EndSession() error
	IsEnded() bool

	GetNextMessageTimestamp() *int64
	SaveNextMessageTimestamp(timestamp int64)
}