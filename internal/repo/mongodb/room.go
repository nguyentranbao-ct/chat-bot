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

// RoomInfo represents room information from partner
type RoomInfo struct {
	Name     string
	Context  string
	Metadata map[string]any
}

// ParticipantInfo represents participant information from partner
type ParticipantInfo struct {
	UserID primitive.ObjectID
	Role   string
}


type RoomMemberRepository interface {
	Create(ctx context.Context, member *models.RoomMember) error
	GetRoomMembers(ctx context.Context, source models.RoomPartner) ([]*models.RoomMember, error)
	GetRoomMembersByRoomID(ctx context.Context, roomID primitive.ObjectID) ([]*models.RoomMember, error)
	GetUserRoomMembers(ctx context.Context, userID primitive.ObjectID) ([]*models.RoomMember, error)
	GetMember(ctx context.Context, source models.RoomPartner, userID primitive.ObjectID) (*models.RoomMember, error)
	GetMemberByRoomID(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) (*models.RoomMember, error)
	FindOrCreateRoom(ctx context.Context, source models.RoomPartner, roomInfo RoomInfo, participants []ParticipantInfo) error
	IncrementUnreadCountByRoomID(ctx context.Context, roomID primitive.ObjectID, excludeUserID primitive.ObjectID) error
	IncrementUnreadCount(ctx context.Context, source models.RoomPartner, excludeUserID primitive.ObjectID) error
	IncrementUnreadCountAndUpdateLastMessage(ctx context.Context, roomID primitive.ObjectID, excludeUserID primitive.ObjectID, content string) error
	MarkAsRead(ctx context.Context, userID primitive.ObjectID, source models.RoomPartner, lastReadMessageID primitive.ObjectID) error
	MarkAsReadByRoomID(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID, lastReadMessageID primitive.ObjectID) error
	UpdateLastMessageForRoom(ctx context.Context, roomID primitive.ObjectID, content string) error
}

type roomMemberRepo struct {
	collection *mongo.Collection
}

func NewRoomMemberRepository(db *DB) RoomMemberRepository {
	return &roomMemberRepo{
		collection: db.Database.Collection("room_members"),
	}
}

func (r *roomMemberRepo) Create(ctx context.Context, member *models.RoomMember) error {
	member.ID = primitive.NewObjectID()
	member.JoinedAt = time.Now()
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, member)
	if err != nil {
		return fmt.Errorf("failed to create room member: %w", err)
	}
	return nil
}

func (r *roomMemberRepo) GetRoomMembers(ctx context.Context, source models.RoomPartner) ([]*models.RoomMember, error) {
	filter := bson.M{
		"source.name":    source.Name,
		"source.room_id": source.RoomID,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	defer cursor.Close(ctx)

	var members []*models.RoomMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, fmt.Errorf("failed to decode members: %w", err)
	}

	return members, nil
}

func (r *roomMemberRepo) GetRoomMembersByRoomID(ctx context.Context, roomID primitive.ObjectID) ([]*models.RoomMember, error) {
	filter := bson.M{"room_id": roomID}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	defer cursor.Close(ctx)

	var members []*models.RoomMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, fmt.Errorf("failed to decode members: %w", err)
	}

	return members, nil
}

func (r *roomMemberRepo) GetUserRoomMembers(ctx context.Context, userID primitive.ObjectID) ([]*models.RoomMember, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.M{"last_message_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get user room members: %w", err)
	}
	defer cursor.Close(ctx)

	var members []*models.RoomMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, fmt.Errorf("failed to decode room members: %w", err)
	}

	return members, nil
}

func (r *roomMemberRepo) GetMember(ctx context.Context, source models.RoomPartner, userID primitive.ObjectID) (*models.RoomMember, error) {
	var member models.RoomMember
	filter := bson.M{
		"source.name":    source.Name,
		"source.room_id": source.RoomID,
		"user_id":        userID,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&member)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return &member, nil
}

func (r *roomMemberRepo) GetMemberByRoomID(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) (*models.RoomMember, error) {
	var member models.RoomMember
	filter := bson.M{
		"room_id": roomID,
		"user_id": userID,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&member)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return &member, nil
}

func (r *roomMemberRepo) FindOrCreateRoom(ctx context.Context, source models.RoomPartner, roomInfo RoomInfo, participants []ParticipantInfo) error {
	// Check if room already exists by looking for any member with this source
	filter := bson.M{
		"source.name":    source.Name,
		"source.room_id": source.RoomID,
	}

	var existingMember models.RoomMember
	err := r.collection.FindOne(ctx, filter).Decode(&existingMember)
	if err == nil {
		// Room already exists
		return nil
	}
	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to check existing room: %w", err)
	}

	// Create room members for all participants
	roomID := primitive.NewObjectID()
	now := time.Now()

	for _, participant := range participants {
		member := &models.RoomMember{
			ID:          primitive.NewObjectID(),
			UserID:      participant.UserID,
			Role:        participant.Role,
			Source:      source,
			RoomID:      roomID,
			RoomName:    roomInfo.Name,
			RoomContext: roomInfo.Context,
			Metadata:    roomInfo.Metadata,
			JoinedAt:    now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if _, err := r.collection.InsertOne(ctx, member); err != nil {
			return fmt.Errorf("failed to create room member: %w", err)
		}
	}

	return nil
}

func (r *roomMemberRepo) IncrementUnreadCount(ctx context.Context, source models.RoomPartner, excludeUserID primitive.ObjectID) error {
	filter := bson.M{
		"source.name":    source.Name,
		"source.room_id": source.RoomID,
		"user_id":        bson.M{"$ne": excludeUserID},
	}
	update := bson.M{
		"$inc": bson.M{"unread_count": 1},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

func (r *roomMemberRepo) MarkAsRead(ctx context.Context, userID primitive.ObjectID, source models.RoomPartner, lastReadMessageID primitive.ObjectID) error {
	filter := bson.M{
		"user_id":        userID,
		"source.name":    source.Name,
		"source.room_id": source.RoomID,
	}
	update := bson.M{
		"$set": bson.M{
			"unread_count": 0,
			"last_read_at": time.Now(),
			"updated_at":   time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *roomMemberRepo) IncrementUnreadCountByRoomID(ctx context.Context, roomID primitive.ObjectID, excludeUserID primitive.ObjectID) error {
	filter := bson.M{
		"room_id": roomID,
		"user_id": bson.M{"$ne": excludeUserID},
	}
	update := bson.M{
		"$inc": bson.M{"unread_count": 1},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

func (r *roomMemberRepo) MarkAsReadByRoomID(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID, lastReadMessageID primitive.ObjectID) error {
	filter := bson.M{
		"room_id": roomID,
		"user_id": userID,
	}
	update := bson.M{
		"$set": bson.M{
			"unread_count": 0,
			"last_read_at": time.Now(),
			"updated_at":   time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *roomMemberRepo) IncrementUnreadCountAndUpdateLastMessage(ctx context.Context, roomID primitive.ObjectID, excludeUserID primitive.ObjectID, content string) error {
	now := time.Now()

	// Single query to increment unread count for others and update last message for all room members
	// Use MongoDB's bulkWrite for atomic operations on different filter conditions
	models := []mongo.WriteModel{
		// Increment unread count for others (exclude sender)
		mongo.NewUpdateManyModel().
			SetFilter(bson.M{
				"room_id": roomID,
				"user_id": bson.M{"$ne": excludeUserID},
			}).
			SetUpdate(bson.M{
				"$inc": bson.M{"unread_count": 1},
				"$set": bson.M{
					"last_message_at":      now,
					"last_message_content": content,
					"updated_at":           now,
				},
			}),
		// Update last message for sender (without incrementing unread count)
		mongo.NewUpdateOneModel().
			SetFilter(bson.M{
				"room_id": roomID,
				"user_id": excludeUserID,
			}).
			SetUpdate(bson.M{
				"$set": bson.M{
					"last_message_at":      now,
					"last_message_content": content,
					"updated_at":           now,
				},
			}),
	}

	_, err := r.collection.BulkWrite(ctx, models)
	return err
}

func (r *roomMemberRepo) UpdateLastMessageForRoom(ctx context.Context, roomID primitive.ObjectID, content string) error {
	filter := bson.M{"room_id": roomID}
	update := bson.M{
		"$set": bson.M{
			"last_message_at":      time.Now(),
			"last_message_content": content,
			"updated_at":           time.Now(),
		},
	}

	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

