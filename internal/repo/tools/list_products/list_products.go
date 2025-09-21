package list_products

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ToolName        = "ListProducts"
	ToolDescription = "List products for a user from their linked external service accounts"
)

// ListProductsInput defines the input arguments for the ListProducts tool
type ListProductsInput struct {
	Limit int `json:"limit,omitempty"`
	Page  int `json:"page,omitempty"`
}

// ListProductsOutput defines the output of the ListProducts tool
type ListProductsOutput struct {
	Products []Product `json:"products"`
	Total    int       `json:"total"`
}

type Tool interface {
	toolsmanager.Tool
}

// tool implements the toolsmanager.Tool interface
type tool struct {
	userRepo          mongodb.UserRepository
	userAttributeRepo mongodb.UserAttributeRepository
	activityRepo      mongodb.ChatActivityRepository
	serviceRegistry   ProductServiceRegistry
}

// NewTool creates a new ListProducts tool instance
func NewTool(
	userRepo mongodb.UserRepository,
	userAttributeRepo mongodb.UserAttributeRepository,
	activityRepo mongodb.ChatActivityRepository,
	serviceRegistry ProductServiceRegistry,
) Tool {
	return &tool{
		userRepo:          userRepo,
		userAttributeRepo: userAttributeRepo,
		activityRepo:      activityRepo,
		serviceRegistry:   serviceRegistry,
	}
}

// Name returns the tool's unique identifier
func (t *tool) Name() string {
	return ToolName
}

// Description returns a human-readable description of what the tool does
func (t *tool) Description() string {
	return ToolDescription
}

// Execute runs the tool with the given arguments and session context
func (t *tool) Execute(ctx context.Context, args interface{}, session toolsmanager.SessionContext) (interface{}, error) {
	// Parse arguments
	var input ListProductsInput
	if err := t.parseArgs(args, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Set defaults
	if input.Limit <= 0 {
		input.Limit = 9
	}
	if input.Page <= 0 {
		input.Page = 1
	}

	// Get seller ID from session context (this is the chotot_id)
	sellerID := session.GetSenderID()
	if sellerID == "" {
		return nil, fmt.Errorf("no seller ID found in session context")
	}

	// Step 1: Map from chotot_id (seller ID) to internal user ID
	// Find user attribute with key="chotot_id" and value=sellerID
	chototIDAttrs, err := t.userAttributeRepo.GetByKey(ctx, "chotot_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get chotot_id attributes: %w", err)
	}

	var internalUserID primitive.ObjectID
	var found bool
	for _, attr := range chototIDAttrs {
		if attr.Value == sellerID {
			internalUserID = attr.UserID
			found = true
			break
		}
	}

	if !found {
		log.Infow(ctx, "No internal user found for chotot_id", "chotot_id", sellerID)
		return &ListProductsOutput{
			Products: []Product{},
			Total:    0,
		}, nil
	}

	// Step 2: Get the chotot_oid attribute for this internal user
	chototOIDAttr, err := t.userAttributeRepo.GetByUserIDAndKey(ctx, internalUserID, "chotot_oid")
	if err != nil {
		return nil, fmt.Errorf("failed to get chotot_oid attribute: %w", err)
	}

	if chototOIDAttr == nil {
		log.Infow(ctx, "No chotot_oid found for internal user", "internal_user_id", internalUserID.Hex())
		return &ListProductsOutput{
			Products: []Product{},
			Total:    0,
		}, nil
	}

	// Get Chotot product service
	service, exists := t.serviceRegistry.GetService("chotot")
	if !exists {
		return nil, fmt.Errorf("chotot product service not registered")
	}

	// Step 3: Fetch products from Chotot service using the chotot_oid value
	products, total, err := service.ListUserProducts(ctx, chototOIDAttr.Value, input.Limit, input.Page)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products from Chotot: %w", err)
	}

	// Log activity
	if err := t.logActivity(ctx, input, session); err != nil {
		log.Errorf(ctx, "Failed to log ListProducts activity: %v", err)
	}

	output := &ListProductsOutput{
		Products: products,
		Total:    total,
	}

	log.Infow(ctx, "Successfully fetched products from Chotot",
		"chotot_id", sellerID,
		"internal_user_id", internalUserID.Hex(),
		"chotot_oid", chototOIDAttr.Value,
		"products_count", len(products),
		"total_available", total)

	return output, nil
}

// GetGenkitTool returns the Firebase Genkit tool definition for AI integration
func (t *tool) GetGenkitTool(session toolsmanager.SessionContext, g *genkit.Genkit) ai.Tool {
	return genkit.DefineTool(g, ToolName, ToolDescription,
		func(toolCtx *ai.ToolContext, input ListProductsInput) (*ListProductsOutput, error) {
			result, err := t.Execute(session.Context(), input, session)
			if err != nil {
				return nil, err
			}

			if output, ok := result.(*ListProductsOutput); ok {
				return output, nil
			}
			return nil, fmt.Errorf("unexpected result type: %T", result)
		})
}

// parseArgs converts interface{} arguments to the expected type
func (t *tool) parseArgs(args interface{}, target interface{}) error {
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal args: %w", err)
	}
	return nil
}


// logActivity logs the tool execution activity
func (t *tool) logActivity(ctx context.Context, input ListProductsInput, session toolsmanager.SessionContext) error {
	sessionID, err := primitive.ObjectIDFromHex(session.GetSessionID())
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityListProducts,
		Data:      input,
	}

	return t.activityRepo.Create(ctx, activity)
}