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

type RoomRepository interface {
	Create(ctx context.Context, room *models.Room) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Room, error)
	GetByPartnerRoomID(ctx context.Context, partnerName, partnerRoomID string) (*models.Room, error)
	GetUserRooms(ctx context.Context, userID primitive.ObjectID) ([]*models.Room, error)
	UpdateLastMessage(ctx context.Context, roomID primitive.ObjectID) error
	GetRoomsWithUnreadCount(ctx context.Context, userID primitive.ObjectID) ([]bson.M, error)
}

type roomRepo struct {
	collection *mongo.Collection
}

func NewRoomRepository(db *DB) RoomRepository {
	return &roomRepo{
		collection: db.Database.Collection("rooms"),
	}
}

func (r *roomRepo) Create(ctx context.Context, room *models.Room) error {
	room.ID = primitive.NewObjectID()
	room.CreatedAt = time.Now()
	room.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, room)
	if err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}
	return nil
}

func (r *roomRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Room, error) {
	var room models.Room
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("room not found")
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	return &room, nil
}

// GetByPartnerRoomID finds a room by partner name and partner room ID
func (r *roomRepo) GetByPartnerRoomID(ctx context.Context, partnerName, partnerRoomID string) (*models.Room, error) {
	var room models.Room
	filter := bson.M{
		"partner.name":    partnerName,
		"partner.room_id": partnerRoomID,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("room not found")
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	return &room, nil
}

func (r *roomRepo) GetUserRooms(ctx context.Context, userID primitive.ObjectID) ([]*models.Room, error) {
	// Join with room_members to get user's rooms
	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "room_members",
				"localField":   "_id",
				"foreignField": "room_id",
				"as":           "members",
			},
		},
		{
			"$match": bson.M{
				"members.user_id":   userID,
				"members.is_active": true,
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

	var rooms []*models.Room
	if err := cursor.All(ctx, &rooms); err != nil {
		return nil, err
	}

	return rooms, nil
}

func (r *roomRepo) UpdateLastMessage(ctx context.Context, roomID primitive.ObjectID) error {
	filter := bson.M{"_id": roomID}
	update := bson.M{
		"$set": bson.M{
			"last_message_at": time.Now(),
			"updated_at":      time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *roomRepo) GetRoomsWithUnreadCount(ctx context.Context, userID primitive.ObjectID) ([]bson.M, error) {
	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "room_members",
				"localField":   "_id",
				"foreignField": "room_id",
				"as":           "members",
			},
		},
		{
			"$match": bson.M{
				"members.user_id":   userID,
				"members.is_active": true,
			},
		},
		{
			"$lookup": bson.M{
				"from": "unread_counts",
				"let":  bson.M{"room_id": "$_id"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$and": []bson.M{
									{"$eq": []interface{}{"$room_id", "$$room_id"}},
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
				"id":              "$_id",
				"partner":         1,
				"name":            1,
				"metadata":        1,
				"context":         1,
				"type":            1,
				"created_at":      1,
				"updated_at":      1,
				"last_message_at": 1,
				"unread_count":    1,
				"_id":             0, // Only _id can be excluded in an inclusion projection
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

type RoomMemberRepository interface {
	Create(ctx context.Context, member *models.RoomMember) error
	GetRoomMembers(ctx context.Context, roomID primitive.ObjectID) ([]*models.RoomMember, error)
	GetMember(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) (*models.RoomMember, error)
	AddMember(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID, role string) error
	RemoveMember(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) error
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

	_, err := r.collection.InsertOne(ctx, member)
	if err != nil {
		return fmt.Errorf("failed to create room member: %w", err)
	}
	return nil
}

func (r *roomMemberRepo) GetRoomMembers(ctx context.Context, roomID primitive.ObjectID) ([]*models.RoomMember, error) {
	filter := bson.M{
		"room_id":   roomID,
		"is_active": true,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	defer cursor.Close(ctx)

	var members []*models.RoomMember
	for cursor.Next(ctx) {
		var member models.RoomMember
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

func (r *roomMemberRepo) GetMember(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) (*models.RoomMember, error) {
	var member models.RoomMember
	err := r.collection.FindOne(ctx, bson.M{
		"room_id": roomID,
		"user_id": userID,
	}).Decode(&member)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return &member, nil
}

func (r *roomMemberRepo) AddMember(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID, role string) error {
	member := &models.RoomMember{
		RoomID:   roomID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	}
	return r.Create(ctx, member)
}

func (r *roomMemberRepo) RemoveMember(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) error {
	filter := bson.M{
		"room_id": roomID,
		"user_id": userID,
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
	GetUnreadCount(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) (*models.UnreadCount, error)
	IncrementUnreadCount(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) error
	MarkAsRead(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID, lastReadMessageID primitive.ObjectID) error
}

type unreadCountRepo struct {
	collection *mongo.Collection
}

func NewUnreadCountRepository(db *DB) UnreadCountRepository {
	return &unreadCountRepo{
		collection: db.Database.Collection("unread_counts"),
	}
}

func (r *unreadCountRepo) GetUnreadCount(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) (*models.UnreadCount, error) {
	var unreadCount models.UnreadCount
	err := r.collection.FindOne(ctx, bson.M{
		"room_id": roomID,
		"user_id": userID,
	}).Decode(&unreadCount)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get unread count: %w", err)
	}
	return &unreadCount, nil
}

func (r *unreadCountRepo) IncrementUnreadCount(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID) error {
	filter := bson.M{
		"room_id": roomID,
		"user_id": userID,
	}

	update := bson.M{
		"$inc": bson.M{"count": 1},
		"$set": bson.M{"updated_at": time.Now()},
		"$setOnInsert": bson.M{
			"room_id": roomID,
			"user_id": userID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *unreadCountRepo) MarkAsRead(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID, lastReadMessageID primitive.ObjectID) error {
	filter := bson.M{
		"room_id": roomID,
		"user_id": userID,
	}

	update := bson.M{
		"$set": bson.M{
			"count":                0,
			"last_read_message_id": lastReadMessageID,
			"updated_at":           time.Now(),
		},
		"$setOnInsert": bson.M{
			"room_id": roomID,
			"user_id": userID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}
