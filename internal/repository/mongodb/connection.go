package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewConnection(ctx context.Context, uri, database string) (*DB, error) {
	clientOptions := options.Client().ApplyURI(uri)
	clientOptions.SetMaxPoolSize(10)
	clientOptions.SetMaxConnIdleTime(30 * time.Second)
	clientOptions.SetTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)

	return &DB{
		client:   client,
		database: db,
	}, nil
}

func (db *DB) Close(ctx context.Context) error {
	return db.client.Disconnect(ctx)
}

func (db *DB) Database() *mongo.Database {
	return db.database
}

func (db *DB) Client() *mongo.Client {
	return db.client
}