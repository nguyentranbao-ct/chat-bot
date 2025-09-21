package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

type ChannelRepository interface {
	Create(ctx context.Context, channel *models.Channel) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Channel, error)
	GetByExternalChannelID(ctx context.Context, externalChannelID string) (*models.Channel, error)
	GetUserChannels(ctx context.Context, userID string) ([]*models.Channel, error)
	UpdateLastMessage(ctx context.Context, channelID primitive.ObjectID) error
	GetChannelsWithUnreadCount(ctx context.Context, userID string) ([]bson.M, error)
}

type channelRepo struct {
	collection *mongo.Collection
}

func NewChannelRepository(db *DB) ChannelRepository {
	return &channelRepo{
		collection: db.Database.Collection("channels"),
	}
}

func (r *channelRepo) Create(ctx context.Context, channel *models.Channel) error {
	channel.ID = primitive.NewObjectID()
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, channel)
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}
	return nil
}

func (r *channelRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Channel, error) {
	var channel models.Channel
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&channel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("channel not found")
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}
	return &channel, nil
}

func (r *channelRepo) GetByExternalChannelID(ctx context.Context, externalChannelID string) (*models.Channel, error) {
	var channel models.Channel
	err := r.collection.FindOne(ctx, bson.M{"external_channel_id": externalChannelID}).Decode(&channel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("channel not found")
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}
	return &channel, nil
}

func (r *channelRepo) GetUserChannels(ctx context.Context, userID string) ([]*models.Channel, error) {
	// Join with channel_members to get user's channels
	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "channel_members",
				"localField":   "_id",
				"foreignField": "channel_id",
				"as":           "members",
			},
		},
		{
			"$match": bson.M{
				"members.user_id":  userID,
				"members.is_active": true,
				"is_archived":      false,
			},
		},
		{
			"$sort": bson.M{"last_message_at": -1},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var channels []*models.Channel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, err
	}

	return channels, nil
}

func (r *channelRepo) UpdateLastMessage(ctx context.Context, channelID primitive.ObjectID) error {
	filter := bson.M{"_id": channelID}
	update := bson.M{
		"$set": bson.M{
			"last_message_at": time.Now(),
			"updated_at":      time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *channelRepo) GetChannelsWithUnreadCount(ctx context.Context, userID string) ([]bson.M, error) {
	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "channel_members",
				"localField":   "_id",
				"foreignField": "channel_id",
				"as":           "members",
			},
		},
		{
			"$match": bson.M{
				"members.user_id":   userID,
				"members.is_active": true,
				"is_archived":       false,
			},
		},
		{
			"$lookup": bson.M{
				"from": "unread_counts",
				"let":  bson.M{"channel_id": "$_id"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$and": []bson.M{
									{"$eq": []interface{}{"$channel_id", "$$channel_id"}},
									{"$eq": []interface{}{"$user_id", userID}},
								},
							},
						},
					},
				},
				"as": "unread_info",
			},
		},
		{
			"$addFields": bson.M{
				"unread_count": bson.M{
					"$ifNull": []interface{}{
						bson.M{"$first": "$unread_info.count"},
						0,
					},
				},
			},
		},
		{
			"$sort": bson.M{"last_message_at": -1},
		},
		{
			"$project": bson.M{
				"members":     0,
				"unread_info": 0,
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

type ChannelMemberRepository interface {
	Create(ctx context.Context, member *models.ChannelMember) error
	GetChannelMembers(ctx context.Context, channelID primitive.ObjectID) ([]*models.ChannelMember, error)
	GetMember(ctx context.Context, channelID primitive.ObjectID, userID string) (*models.ChannelMember, error)
	AddMember(ctx context.Context, channelID primitive.ObjectID, userID, role string) error
	RemoveMember(ctx context.Context, channelID primitive.ObjectID, userID string) error
}

type channelMemberRepo struct {
	collection *mongo.Collection
}

func NewChannelMemberRepository(db *DB) ChannelMemberRepository {
	return &channelMemberRepo{
		collection: db.Database.Collection("channel_members"),
	}
}

func (r *channelMemberRepo) Create(ctx context.Context, member *models.ChannelMember) error {
	member.ID = primitive.NewObjectID()
	member.JoinedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, member)
	if err != nil {
		return fmt.Errorf("failed to create channel member: %w", err)
	}
	return nil
}

func (r *channelMemberRepo) GetChannelMembers(ctx context.Context, channelID primitive.ObjectID) ([]*models.ChannelMember, error) {
	filter := bson.M{
		"channel_id": channelID,
		"is_active":  true,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel members: %w", err)
	}
	defer cursor.Close(ctx)

	var members []*models.ChannelMember
	for cursor.Next(ctx) {
		var member models.ChannelMember
		if err := cursor.Decode(&member); err != nil {
			return nil, fmt.Errorf("failed to decode member: %w", err)
		}
		members = append(members, &member)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return members, nil
}

func (r *channelMemberRepo) GetMember(ctx context.Context, channelID primitive.ObjectID, userID string) (*models.ChannelMember, error) {
	var member models.ChannelMember
	err := r.collection.FindOne(ctx, bson.M{
		"channel_id": channelID,
		"user_id":    userID,
	}).Decode(&member)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return &member, nil
}

func (r *channelMemberRepo) AddMember(ctx context.Context, channelID primitive.ObjectID, userID, role string) error {
	member := &models.ChannelMember{
		ChannelID: channelID,
		UserID:    userID,
		Role:      role,
		JoinedAt:  time.Now(),
		IsActive:  true,
	}
	return r.Create(ctx, member)
}

func (r *channelMemberRepo) RemoveMember(ctx context.Context, channelID primitive.ObjectID, userID string) error {
	filter := bson.M{
		"channel_id": channelID,
		"user_id":    userID,
	}
	update := bson.M{
		"$set": bson.M{
			"is_active": false,
			"left_at":   time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

type UnreadCountRepository interface {
	GetUnreadCount(ctx context.Context, channelID primitive.ObjectID, userID string) (*models.UnreadCount, error)
	IncrementUnreadCount(ctx context.Context, channelID primitive.ObjectID, userID string) error
	MarkAsRead(ctx context.Context, channelID primitive.ObjectID, userID string, lastReadMessageID primitive.ObjectID) error
}

type unreadCountRepo struct {
	collection *mongo.Collection
}

func NewUnreadCountRepository(db *DB) UnreadCountRepository {
	return &unreadCountRepo{
		collection: db.Database.Collection("unread_counts"),
	}
}

func (r *unreadCountRepo) GetUnreadCount(ctx context.Context, channelID primitive.ObjectID, userID string) (*models.UnreadCount, error) {
	var unreadCount models.UnreadCount
	err := r.collection.FindOne(ctx, bson.M{
		"channel_id": channelID,
		"user_id":    userID,
	}).Decode(&unreadCount)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get unread count: %w", err)
	}
	return &unreadCount, nil
}

func (r *unreadCountRepo) IncrementUnreadCount(ctx context.Context, channelID primitive.ObjectID, userID string) error {
	filter := bson.M{
		"channel_id": channelID,
		"user_id":    userID,
	}

	update := bson.M{
		"$inc": bson.M{"count": 1},
		"$set": bson.M{"updated_at": time.Now()},
		"$setOnInsert": bson.M{
			"channel_id": channelID,
			"user_id":    userID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *unreadCountRepo) MarkAsRead(ctx context.Context, channelID primitive.ObjectID, userID string, lastReadMessageID primitive.ObjectID) error {
	filter := bson.M{
		"channel_id": channelID,
		"user_id":    userID,
	}

	update := bson.M{
		"$set": bson.M{
			"count":                  0,
			"last_read_message_id":   lastReadMessageID,
			"updated_at":             time.Now(),
		},
		"$setOnInsert": bson.M{
			"channel_id": channelID,
			"user_id":    userID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}