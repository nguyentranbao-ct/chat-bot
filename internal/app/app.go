package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/handler"
	"github.com/nguyentranbao-ct/chat-bot/internal/llm"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
	"go.uber.org/fx"
)

func NewApp() *fx.App {
	return fx.New(
		fx.Provide(
			config.Load,
			NewMongoDB,
			NewRepositories,
			NewChatAPIClient,
			NewToolsManager,
			NewGenkitService,
			NewMessageUsecase,
			NewMessageHandler,
			NewEchoServer,
		),
		fx.Invoke(StartServer),
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
