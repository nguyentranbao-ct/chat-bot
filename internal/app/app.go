package app

import (
	"context"

	"github.com/carousell/ct-go/pkg/logger"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chotot"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/end_session"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/fetch_messages"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/purchase_intent"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/reply_message"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"github.com/nguyentranbao-ct/chat-bot/internal/server"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
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

			server.NewHandler,

			usecase.NewLLMUsecase,
			usecase.NewMessageUsecase,
			usecase.NewWhitelistService,
			usecase.NewUserUsecase,

			mongodb.NewChatActivityRepository,
			mongodb.NewChatModeRepository,
			mongodb.NewChatSessionRepository,
			mongodb.NewPurchaseIntentRepository,
			mongodb.NewUserRepository,
			mongodb.NewUserAttributeRepository,

			chatapi.NewChatAPIClient,
			chotot.NewClient,
			list_products.NewProductServiceRegistry,

			toolsmanager.NewToolsManager,

			end_session.NewTool,
			purchase_intent.NewTool,
			fetch_messages.NewTool,
			reply_message.NewTool,
			list_products.NewTool,
		),
		fx.Supply(conf),
		fx.Invoke(InitializeUsers),
		fx.Invoke(InitializeProductServices),
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

// InitializeProductServices registers all product services with the registry using fx lifecycle
func InitializeProductServices(
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

// InitializeUsers initializes default users and attributes on startup
func InitializeUsers(
	lc fx.Lifecycle,
	userRepo mongodb.UserRepository,
	userAttrRepo mongodb.UserAttributeRepository,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return usecase.AutoMigrateUsers(userRepo, userAttrRepo)
		},
	})
}
