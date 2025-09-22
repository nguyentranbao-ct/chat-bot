package app

import (
	"context"
	"fmt"

	"github.com/carousell/ct-go/pkg/logger"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chotot"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/internal_api"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/partners"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/socket"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/end_session"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/fetch_messages"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/purchase_intent"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/reply_message"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"github.com/nguyentranbao-ct/chat-bot/internal/server"
	"github.com/nguyentranbao-ct/chat-bot/internal/setup"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap/zapcore"
)

func Invoke(funcs ...any) *fx.App {
	log := logger.MustNamed("app")
	conf := config.MustLoad()
	log.Debugw("config loaded", log.Reflect("config", conf))
	return fx.New(
		fx.WithLogger(func() fxevent.Logger {
			l := &fxevent.ZapLogger{
				Logger: log.Unwrap().Desugar(),
			}
			l.UseLogLevel(zapcore.DebugLevel)
			return l
		}),
		fx.Provide(
			newGenkitClient,
			newMongoDB,
			newJWTSecret,
			newSocketBroadcaster,

			// Controllers
			server.NewHandler,
			server.NewAuthController,
			server.NewChatController,

			// Use Cases
			usecase.NewLLMUsecase,
			usecase.NewWhitelistService,
			usecase.NewUserUsecase,
			usecase.NewAuthUseCase,
			usecase.NewChatUseCase,
			usecase.NewLLMUsecaseV2,

			// Repositories
			mongodb.NewChatActivityRepository,
			mongodb.NewChatModeRepository,
			mongodb.NewChatSessionRepository,
			mongodb.NewPurchaseIntentRepository,
			mongodb.NewUserRepository,
			mongodb.NewUserAttributeRepository,
			mongodb.NewAuthTokenRepository,
			mongodb.NewChannelRepository,
			mongodb.NewChannelMemberRepository,
			mongodb.NewChatMessageRepository,
			mongodb.NewMessageEventRepository,
			mongodb.NewUnreadCountRepository,
			mongodb.NewMessageDedupRepository,

			// repo clients
			chatapi.NewChatAPIClient,
			chotot.NewClient,
			list_products.NewProductServiceRegistry,
			socket.NewClient,
			internal_api.NewClient,

			// Partner System
			partners.NewPartnerRegistry,
			partners.NewChototPartner,

			// Tools Manager
			toolsmanager.NewToolsManager,

			// Tools
			end_session.NewTool,
			purchase_intent.NewTool,
			fetch_messages.NewTool,
			reply_message.NewTool,
			list_products.NewTool,
		),
		fx.Supply(conf),
		fx.Invoke(setup.SetupUsers),
		fx.Invoke(setup.SetupChannels),
		fx.Invoke(setup.SetupChatModes),
		fx.Invoke(initializeProductServices),
		fx.Invoke(initializeLLMTools),
		fx.Invoke(initializePartners),
		fx.Invoke(funcs...),
	)
}

func newGenkitClient(cfg *config.Config) (*genkit.Genkit, error) {
	ctx := context.Background()
	googleAI := &googlegenai.GoogleAI{
		APIKey: cfg.LLM.GoogleAIAPIKey,
	}
	return genkit.Init(ctx, genkit.WithPlugins(googleAI)), nil
}

func newJWTSecret(cfg *config.Config) string {
	return cfg.JWT.Secret
}

func newSocketBroadcaster(client *socket.Client) usecase.SocketBroadcaster {
	return socket.NewBroadcaster(client)
}

// initializeProductServices registers all product services with the registry using fx lifecycle
func initializeProductServices(
	lc fx.Lifecycle,
	client chotot.Client,
	registry list_products.ProductServiceRegistry,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Register chotot product service
			chototService := chotot.NewProductService(client)
			registry.RegisterService("chotot", chototService)
			return nil
		},
	})
}

func initializeLLMTools(
	toolsManager toolsmanager.ToolsManager,
	endSessionTool end_session.Tool,
	fetchMessagesTool fetch_messages.Tool,
	replyMessageTool reply_message.Tool,
	purchaseIntentTool purchase_intent.Tool,
	listProductsTool list_products.Tool,
) {
	util.PanicOnError(
		"register tools",
		toolsManager.AddTool(endSessionTool),
		toolsManager.AddTool(fetchMessagesTool),
		toolsManager.AddTool(replyMessageTool),
		toolsManager.AddTool(purchaseIntentTool),
		toolsManager.AddTool(listProductsTool),
	)
}

// initializePartners registers all partner implementations with the registry
func initializePartners(
	lc fx.Lifecycle,
	registry *partners.PartnerRegistry,
	chototPartner *partners.ChototPartner,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Register Chotot partner
			if err := registry.RegisterPartner(chototPartner); err != nil {
				return fmt.Errorf("failed to register Chotot partner: %w", err)
			}
			return nil
		},
	})
}
