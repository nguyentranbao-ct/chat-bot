package mongodb

import (
	"context"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

// MigrationRepository handles database migrations
type MigrationRepository interface {
	MigrateChannelsToVendorModel(ctx context.Context) error
	MigrateChatMessagesToRemoveExternalChannelID(ctx context.Context) error
	GetMigrationStatus(ctx context.Context, migrationName string) (*MigrationStatus, error)
	SetMigrationStatus(ctx context.Context, migrationName string, status string, result *MigrationResult) error
}

type migrationRepo struct {
	db *DB
}

// MigrationStatus tracks the status of database migrations
type MigrationStatus struct {
	ID          string           `bson:"_id" json:"id"`
	Name        string           `bson:"name" json:"name"`
	Status      string           `bson:"status" json:"status"` // "pending", "running", "completed", "failed"
	StartedAt   *time.Time       `bson:"started_at" json:"started_at"`
	CompletedAt *time.Time       `bson:"completed_at" json:"completed_at"`
	Result      *MigrationResult `bson:"result,omitempty" json:"result,omitempty"`
	CreatedAt   time.Time        `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `bson:"updated_at" json:"updated_at"`
}

// MigrationResult contains the results of a migration
type MigrationResult struct {
	RecordsProcessed int      `bson:"records_processed" json:"records_processed"`
	RecordsUpdated   int      `bson:"records_updated" json:"records_updated"`
	RecordsSkipped   int      `bson:"records_skipped" json:"records_skipped"`
	Errors           []string `bson:"errors,omitempty" json:"errors,omitempty"`
	Duration         string   `bson:"duration" json:"duration"`
}

func NewMigrationRepository(db *DB) MigrationRepository {
	return &migrationRepo{
		db: db,
	}
}

// MigrateChannelsToVendorModel migrates existing channels from ExternalChannelID to Vendor model
func (r *migrationRepo) MigrateChannelsToVendorModel(ctx context.Context) error {
	migrationName := "channels_to_vendor_model"

	// Check if migration already completed
	status, err := r.GetMigrationStatus(ctx, migrationName)
	if err == nil && status.Status == "completed" {
		log.Infow(ctx, "Migration already completed", "migration", migrationName)
		return nil
	}

	// Set migration as running
	startTime := time.Now()
	if err := r.SetMigrationStatus(ctx, migrationName, "running", nil); err != nil {
		return fmt.Errorf("failed to set migration status: %w", err)
	}

	log.Infow(ctx, "Starting channel migration to vendor model", "migration", migrationName)

	collection := r.db.Database.Collection("channels")
	result := &MigrationResult{}

	// Find all channels that still have external_channel_id but no vendor field
	filter := bson.M{
		"external_channel_id": bson.M{"$exists": true, "$ne": ""},
		"vendor":              bson.M{"$exists": false},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return r.completeMigrationWithError(ctx, migrationName, startTime, fmt.Errorf("failed to find channels: %w", err))
	}
	defer cursor.Close(ctx)

	var batch []mongo.WriteModel
	batchSize := 100

	for cursor.Next(ctx) {
		result.RecordsProcessed++

		var oldChannel struct {
			ID                string `bson:"_id"`
			ExternalChannelID string `bson:"external_channel_id"`
			ItemName          string `bson:"item_name,omitempty"`
			ItemPrice         string `bson:"item_price,omitempty"`
		}

		if err := cursor.Decode(&oldChannel); err != nil {
			errMsg := fmt.Sprintf("failed to decode channel %s: %v", oldChannel.ID, err)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		// Create vendor object
		vendor := models.ChannelVendor{
			ChannelID: oldChannel.ExternalChannelID,
			Name:      "chotot", // Default all existing channels to Chotot
		}

		// Create metadata with ItemName and ItemPrice if they exist
		metadata := make(map[string]any)
		if oldChannel.ItemName != "" {
			metadata["item_name"] = oldChannel.ItemName
		}
		if oldChannel.ItemPrice != "" {
			metadata["item_price"] = oldChannel.ItemPrice
		}

		// Prepare update operation
		update := bson.M{
			"$set": bson.M{
				"vendor":     vendor,
				"metadata":   metadata,
				"updated_at": time.Now(),
			},
			"$unset": bson.M{
				"external_channel_id": "",
				"item_name":           "",
				"item_price":          "",
			},
		}

		// Add to batch
		batch = append(batch, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": oldChannel.ID}).
			SetUpdate(update))

		// Execute batch when it reaches the limit
		if len(batch) >= batchSize {
			if err := r.executeBatch(ctx, collection, batch, result); err != nil {
				log.Errorw(ctx, "Failed to execute batch", "error", err)
			}
			batch = batch[:0] // Clear batch
		}
	}

	// Execute remaining batch
	if len(batch) > 0 {
		if err := r.executeBatch(ctx, collection, batch, result); err != nil {
			log.Errorw(ctx, "Failed to execute final batch", "error", err)
		}
	}

	if err := cursor.Err(); err != nil {
		return r.completeMigrationWithError(ctx, migrationName, startTime, fmt.Errorf("cursor error: %w", err))
	}

	// Complete migration
	duration := time.Since(startTime)
	result.Duration = duration.String()

	if err := r.SetMigrationStatus(ctx, migrationName, "completed", result); err != nil {
		log.Errorw(ctx, "Failed to set migration completion status", "error", err)
	}

	log.Infow(ctx, "Channel migration completed successfully",
		"migration", migrationName,
		"processed", result.RecordsProcessed,
		"updated", result.RecordsUpdated,
		"skipped", result.RecordsSkipped,
		"errors", len(result.Errors),
		"duration", result.Duration)

	return nil
}

// MigrateChatMessagesToRemoveExternalChannelID removes external_channel_id field from chat messages
func (r *migrationRepo) MigrateChatMessagesToRemoveExternalChannelID(ctx context.Context) error {
	migrationName := "remove_external_channel_id_from_messages"

	// Check if migration already completed
	status, err := r.GetMigrationStatus(ctx, migrationName)
	if err == nil && status.Status == "completed" {
		log.Infow(ctx, "Migration already completed", "migration", migrationName)
		return nil
	}

	// Set migration as running
	startTime := time.Now()
	if err := r.SetMigrationStatus(ctx, migrationName, "running", nil); err != nil {
		return fmt.Errorf("failed to set migration status: %w", err)
	}

	log.Infow(ctx, "Starting chat message migration to remove external_channel_id", "migration", migrationName)

	collection := r.db.Database.Collection("messages")

	// Remove external_channel_id field from all messages
	updateResult, err := collection.UpdateMany(ctx,
		bson.M{"external_channel_id": bson.M{"$exists": true}},
		bson.M{"$unset": bson.M{"external_channel_id": ""}})

	duration := time.Since(startTime)
	result := &MigrationResult{
		RecordsProcessed: int(updateResult.MatchedCount),
		RecordsUpdated:   int(updateResult.ModifiedCount),
		Duration:         duration.String(),
	}

	if err != nil {
		return r.completeMigrationWithError(ctx, migrationName, startTime, err)
	}

	// Complete migration
	if err := r.SetMigrationStatus(ctx, migrationName, "completed", result); err != nil {
		log.Errorw(ctx, "Failed to set migration completion status", "error", err)
	}

	log.Infow(ctx, "Chat message migration completed successfully",
		"migration", migrationName,
		"processed", result.RecordsProcessed,
		"updated", result.RecordsUpdated,
		"duration", result.Duration)

	return nil
}

func (r *migrationRepo) executeBatch(ctx context.Context, collection *mongo.Collection, batch []mongo.WriteModel, result *MigrationResult) error {
	if len(batch) == 0 {
		return nil
	}

	bulkResult, err := collection.BulkWrite(ctx, batch, options.BulkWrite().SetOrdered(false))
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("bulk write error: %v", err))
		return err
	}

	result.RecordsUpdated += int(bulkResult.ModifiedCount)
	result.RecordsSkipped += len(batch) - int(bulkResult.ModifiedCount)

	return nil
}

func (r *migrationRepo) completeMigrationWithError(ctx context.Context, migrationName string, startTime time.Time, err error) error {
	duration := time.Since(startTime)
	result := &MigrationResult{
		Duration: duration.String(),
		Errors:   []string{err.Error()},
	}

	if setErr := r.SetMigrationStatus(ctx, migrationName, "failed", result); setErr != nil {
		log.Errorw(ctx, "Failed to set migration failure status", "error", setErr)
	}

	return err
}

func (r *migrationRepo) GetMigrationStatus(ctx context.Context, migrationName string) (*MigrationStatus, error) {
	collection := r.db.Database.Collection("migrations")

	var status MigrationStatus
	err := collection.FindOne(ctx, bson.M{"name": migrationName}).Decode(&status)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("migration status not found: %s", migrationName)
		}
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	return &status, nil
}

func (r *migrationRepo) SetMigrationStatus(ctx context.Context, migrationName string, status string, result *MigrationResult) error {
	collection := r.db.Database.Collection("migrations")

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"name":       migrationName,
			"status":     status,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	if status == "running" {
		update["$set"].(bson.M)["started_at"] = now
	} else if status == "completed" || status == "failed" {
		update["$set"].(bson.M)["completed_at"] = now
		if result != nil {
			update["$set"].(bson.M)["result"] = result
		}
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, bson.M{"name": migrationName}, update, opts)
	if err != nil {
		return fmt.Errorf("failed to set migration status: %w", err)
	}

	return nil
}
