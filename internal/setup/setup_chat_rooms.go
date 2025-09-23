package setup

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/yaml.v3"
)

//go:embed data/default_chat_rooms.yaml
var defaultRoomsData []byte

func SetupRooms(userRepo mongodb.UserRepository, roomMemberRepo mongodb.RoomMemberRepository, messageRepo mongodb.ChatMessageRepository) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load and create default rooms
	var defaultRooms []DefaultRoom
	if err := yaml.Unmarshal(defaultRoomsData, &defaultRooms); err != nil {
		return fmt.Errorf("failed to unmarshal default rooms: %w", err)
	}

	log.Debugw(ctx, "Loaded rooms from YAML", "count", len(defaultRooms))

	// Create rooms if they don't exist
	for _, defaultRoom := range defaultRooms {
		source := models.RoomPartner{
			RoomID: defaultRoom.ExternalRoomID,
			Name:   "chotot", // Default partner for demo rooms
		}

		// Check if room already exists by looking for members with this source
		existingMembers, err := roomMemberRepo.GetRoomMembers(ctx, source)
		if err == nil && len(existingMembers) > 0 {
			log.Debugw(ctx, "Room already exists", "partner_room_id", defaultRoom.ExternalRoomID)
			continue
		}

		// Find owner user by email
		ownerUser, err := userRepo.GetByEmail(ctx, defaultRoom.OwnerEmail)
		if err != nil || ownerUser == nil {
			log.Warnw(ctx, "Owner user not found for room", "owner_email", defaultRoom.OwnerEmail, "room_id", defaultRoom.ExternalRoomID)
			continue
		}

		now := time.Now()

		// Create metadata from item name and price
		metadata := make(map[string]any)
		if defaultRoom.ItemName != "" {
			metadata["item_name"] = defaultRoom.ItemName
		}
		if defaultRoom.ItemPrice != "" {
			metadata["item_price"] = defaultRoom.ItemPrice
		}

		// Create room member for the owner using the new enhanced structure
		member := &models.RoomMember{
			UserID: ownerUser.ID,
			Role:   "merchant",

			// Room information (denormalized)
			Source:      source,
			RoomID:      primitive.NewObjectID(), // Generate a new room ID
			RoomName:    defaultRoom.Name,
			RoomContext: defaultRoom.Context,
			Metadata:    metadata,

			// Timestamps
			JoinedAt:  now,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := roomMemberRepo.Create(ctx, member); err != nil {
			return fmt.Errorf("failed to create room member for '%s': %w", defaultRoom.ExternalRoomID, err)
		}
		log.Infow(ctx, "Created room member", "partner_room_id", defaultRoom.ExternalRoomID, "user_id", defaultRoom.OwnerEmail)
	}
	return nil
}
