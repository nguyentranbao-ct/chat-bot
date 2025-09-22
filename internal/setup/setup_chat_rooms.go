package setup

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"gopkg.in/yaml.v3"
)

//go:embed data/default_chat_rooms.yaml
var defaultRoomsData []byte

func SetupRooms(userRepo mongodb.UserRepository, roomRepo mongodb.RoomRepository, roomMemberRepo mongodb.RoomMemberRepository, messageRepo mongodb.ChatMessageRepository) error {
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
		// Use GetByPartnerRoomID instead of deprecated GetByExternalRoomID for consistency
		existingRoom, err := roomRepo.GetByPartnerRoomID(ctx, "chotot", defaultRoom.ExternalRoomID)
		if err != nil && existingRoom == nil {
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

			room := &models.Room{
				Source: models.RoomPartner{
					RoomID: defaultRoom.ExternalRoomID,
					Name:   "chotot", // Default partner for demo rooms
				},
				Name:      defaultRoom.Name,
				Context:   defaultRoom.Context,
				Metadata:  metadata,
				CreatedAt: now,
				UpdatedAt: now,
			}

			if err := roomRepo.Create(ctx, room); err != nil {
				return fmt.Errorf("failed to create room '%s': %w", defaultRoom.ExternalRoomID, err)
			}
			log.Infow(ctx, "Created default room", "partner_room_id", defaultRoom.ExternalRoomID, "name", defaultRoom.Name)

			// Create room member for the owner
			member := &models.RoomMember{
				RoomID:   room.ID,
				UserID:   ownerUser.ID, // Using email as user ID for simplicity
				Role:     "seller",
				JoinedAt: now,
			}

			if err := roomMemberRepo.Create(ctx, member); err != nil {
				return fmt.Errorf("failed to create room member for '%s': %w", defaultRoom.ExternalRoomID, err)
			}
			log.Infow(ctx, "Created room member", "partner_room_id", defaultRoom.ExternalRoomID, "user_id", defaultRoom.OwnerEmail)
		} else {
			log.Debugw(ctx, "Room already exists", "partner_room_id", defaultRoom.ExternalRoomID)
		}
	}
	return nil
}
