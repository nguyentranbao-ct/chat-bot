package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/carousell/ct-go/pkg/logger"
	"github.com/labstack/echo/v4"
	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/handler"
	"github.com/nguyentranbao-ct/chat-bot/internal/kafka"
	"github.com/nguyentranbao-ct/chat-bot/internal/llm"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/service"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap/zapcore"
)

func NewApp() *fx.App {
	log := logger.MustNamed("app")
	return fx.New(
		fx.WithLogger(func() fxevent.Logger {
			l := &fxevent.ZapLogger{
				Logger: log.Unwrap().Desugar(),
			}
			l.UseLogLevel(zapcore.DebugLevel)
			return l
		}),
		fx.Provide(
			config.Load,
			NewMongoDB,
			NewRepositories,
			NewChatAPIClient,
			NewToolsManager,
			NewGenkitService,
			NewMessageUsecase,
			NewMessageHandler,
			NewWhitelistService,
			NewKafkaMessageHandler,
			kafka.NewConsumer,
			NewEchoServer,
			NewChatModeInitializer,
		),
		fx.Invoke(StartServer, StartKafkaConsumer, InitializeDefaultChatModes),
	)
}

type Repositories struct {
	ChatMode       repository.ChatModeRepository
	Session        repository.ChatSessionRepository
	Activity       repository.ChatActivityRepository
	PurchaseIntent repository.PurchaseIntentRepository
}

func NewMongoDB(cfg *config.Config) (*mongodb.DB, error) {
	return mongodb.NewConnection(context.Background(), cfg.Database.URI, cfg.Database.Database)
}

func NewRepositories(db *mongodb.DB) *Repositories {
	return &Repositories{
		ChatMode:       mongodb.NewChatModeRepository(db),
		Session:        mongodb.NewChatSessionRepository(db),
		Activity:       mongodb.NewChatActivityRepository(db),
		PurchaseIntent: mongodb.NewPurchaseIntentRepository(db),
	}
}

func NewChatAPIClient(cfg *config.Config) client.ChatAPIClient {
	return client.NewChatAPIClient(&cfg.ChatAPI)
}

func NewToolsManager(
	chatAPIClient client.ChatAPIClient,
	repos *Repositories,
) *llm.ToolsManager {
	return llm.NewToolsManager(
		chatAPIClient,
		repos.Session,
		repos.Activity,
		repos.PurchaseIntent,
	)
}

func NewGenkitService(cfg *config.Config, toolsManager *llm.ToolsManager) (*llm.GenkitService, error) {
	return llm.NewGenkitService(cfg, toolsManager)
}

func NewMessageUsecase(
	repos *Repositories,
	chatAPIClient client.ChatAPIClient,
	genkitService *llm.GenkitService,
) usecase.MessageUsecase {
	return usecase.NewMessageUsecase(
		repos.ChatMode,
		repos.Session,
		repos.Activity,
		chatAPIClient,
		genkitService,
	)
}

func NewMessageHandler(messageUsecase usecase.MessageUsecase) *handler.MessageHandler {
	return handler.NewMessageHandler(messageUsecase)
}

func NewEchoServer(messageHandler *handler.MessageHandler) *echo.Echo {
	e := echo.New()
	handler.SetupMiddleware(e)
	handler.SetupRoutes(e, messageHandler)
	return e
}

func NewWhitelistService(cfg *config.Config) service.WhitelistService {
	return service.NewWhitelistService(&cfg.Kafka)
}

func NewChatModeInitializer(repos *Repositories) service.ChatModeInitializer {
	return service.NewChatModeInitializer(repos.ChatMode)
}

func NewKafkaMessageHandler(messageUsecase usecase.MessageUsecase) kafka.MessageHandler {
	return kafka.NewMessageHandler(messageUsecase)
}


func StartServer(lc fx.Lifecycle, e *echo.Echo, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
				if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
					panic(fmt.Sprintf("Failed to start server: %v", err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return e.Shutdown(ctx)
		},
	})
}

func StartKafkaConsumer(lc fx.Lifecycle, consumer kafka.Consumer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := consumer.Start(ctx); err != nil {
					panic(fmt.Sprintf("Failed to start Kafka consumer: %v", err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return consumer.Stop()
		},
	})
}

func InitializeDefaultChatModes(lc fx.Lifecycle, initializer service.ChatModeInitializer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := initializer.InitializeDefaultChatModes(ctx); err != nil {
				return fmt.Errorf("failed to initialize default chat modes: %w", err)
			}
			return nil
		},
	})
}
