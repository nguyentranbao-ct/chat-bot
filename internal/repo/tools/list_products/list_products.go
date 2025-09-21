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
	UserID string `json:"user_id"`
	Limit  int    `json:"limit,omitempty"`
	Page   int    `json:"page,omitempty"`
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

	// Get user by ID
	userID, err := primitive.ObjectIDFromHex(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := t.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user attributes with "link_id" tag to find linked services
	linkAttrs, err := t.userAttributeRepo.GetByUserIDAndTags(ctx, userID, []string{"link_id"})
	if err != nil {
		return nil, fmt.Errorf("failed to get user link attributes: %w", err)
	}

	if len(linkAttrs) == 0 {
		log.Infow(ctx, "No linked services found for user", "user_id", input.UserID)
		return &ListProductsOutput{
			Products: []Product{},
			Total:    0,
		}, nil
	}

	// Aggregate products from all linked services
	var allProducts []Product
	totalCount := 0

	for _, attr := range linkAttrs {
		// Determine service type from tags
		serviceType := t.determineServiceType(attr.Tags)
		if serviceType == "" {
			log.Warnw(ctx, "Unable to determine service type", "tags", attr.Tags)
			continue
		}

		// Get product service for this type
		service, exists := t.serviceRegistry.GetService(serviceType)
		if !exists {
			log.Warnw(ctx, "No product service registered", "service_type", serviceType)
			continue
		}

		// Fetch products from this service
		products, total, err := service.ListUserProducts(ctx, attr.Value, input.Limit, input.Page)
		if err != nil {
			log.Errorw(ctx, "Failed to fetch products from service",
				"service_type", serviceType,
				"user_external_id", attr.Value,
				"error", err)
			continue
		}

		allProducts = append(allProducts, products...)
		totalCount += total
	}

	// Log activity
	if err := t.logActivity(ctx, input, session); err != nil {
		log.Errorf(ctx, "Failed to log ListProducts activity: %v", err)
	}

	output := &ListProductsOutput{
		Products: allProducts,
		Total:    totalCount,
	}

	log.Infow(ctx, "Successfully fetched products",
		"user_id", input.UserID,
		"user_email", user.Email,
		"products_count", len(allProducts),
		"total_available", totalCount)

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

// determineServiceType determines the service type from tags
func (t *tool) determineServiceType(tags []string) string {
	for _, tag := range tags {
		if tag == "chotot" {
			return "chotot"
		}
		// Add more service types as needed
	}
	return ""
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