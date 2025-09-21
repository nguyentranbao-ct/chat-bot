package mongodb

import (
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventsRepo interface {
	IRepository[models.Event]
}

type eventsRepo struct {
	baseRepo[models.Event]
}

func NewEventsRepo(
	conf *config.Config,
	dbc *mongo.Database,
) EventsRepo {
	r := &eventsRepo{
		baseRepo: newBaseRepo[models.Event](dbc),
	}
	return r
}
