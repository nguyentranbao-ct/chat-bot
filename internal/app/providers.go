package app

import (
	"context"
	"fmt"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/fx"
)

func newMongoDB(lc fx.Lifecycle, cfg *config.Config) (*mongodb.DB, error) {
	opts := options.Client().
		SetAppName("chat-bot").
		SetDirect(cfg.Database.Direct).
		SetHosts(cfg.Database.Hosts)

	if cfg.Database.Username != "" {
		opts.SetAuth(options.Credential{
			Username:      cfg.Database.Username,
			Password:      cfg.Database.Password,
			AuthSource:    cfg.Database.AuthDB,
			AuthMechanism: "SCRAM-SHA-1",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("init mongo client: %w", err)
	}

	mongoDB := mongoClient.Database(cfg.Database.Database)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return mongoClient.Ping(ctx, nil)
		},
		OnStop: func(ctx context.Context) error {
			return mongoClient.Disconnect(ctx)
		},
	})

	return &mongodb.DB{
		Client:   mongoClient,
		Database: mongoDB,
	}, nil
}
