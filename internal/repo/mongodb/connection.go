package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewConnection(ctx context.Context, host, port, username, password, database string) (*DB, error) {
	hostAddr := fmt.Sprintf("%s:%s", host, port)

	clientOptions := options.Client().
		SetAppName("chat-bot").
		SetHosts([]string{hostAddr}).
		SetMaxPoolSize(10).
		SetMaxConnIdleTime(30 * time.Second).
		SetTimeout(10 * time.Second).
		SetDirect(true)

	// Only set auth if password is provided
	if password != "" {
		clientOptions.SetAuth(options.Credential{
			AuthSource: "admin",
			Username:   username,
			Password:   password,
		})
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)

	return &DB{
		Client:   client,
		Database: db,
	}, nil
}

func (db *DB) Close(ctx context.Context) error {
	return db.Client.Disconnect(ctx)
}

func (db *DB) GetDatabase() *mongo.Database {
	return db.Database
}

func (db *DB) GetClient() *mongo.Client {
	return db.Client
}
