package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap/zapcore"
)

func NewApp() *fx.App {
	log := logger.MustNamed("app")
	conf := config.MustLoad()
	log.Infow("Configuration loaded", log.Reflect("config", conf))
	return fx.New(
		fx.WithLogger(func() fxevent.Logger {
			l := &fxevent.ZapLogger{
				Logger: log.Unwrap().Desugar(),
			}
			l.UseLogLevel(zapcore.DebugLevel)
			return l
		}),
		fx.Supply(conf),
		fx.Provide(
			NewKafkaConfig,
			NewMongoDB,
			NewRepositories,
			client.NewChatAPIClient,
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

func NewKafkaConfig(cfg *config.Config) *config.KafkaConfig {
	return &cfg.Kafka
}

func NewMongoDB(lc fx.Lifecycle, cfg *config.Config) (*mongodb.DB, error) {
	opts := options.Client().
		SetAppName("chat-bot").
		SetDirect(cfg.Database.Direct).
		SetHosts(cfg.Database.Hosts)

	if cfg.Database.Username != "" {
		opts.SetAuth(options.Credential{
			Username:      cfg.Database.Username,
			Password:      cfg.Database.Password,
			AuthSource:    cfg.Database.AuthDB,
			AuthMechanism: "SCRAM-SHA-1",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("init mongo client: %w", err)
	}

	mongoDB := mongoClient.Database(cfg.Database.Database)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return mongoClient.Ping(ctx, nil)
		},
		OnStop: func(ctx context.Context) error {
			return mongoClient.Disconnect(ctx)
		},
	})

	return &mongodb.DB{
		Client:   mongoClient,
		Database: mongoDB,
	}, nil
}

func NewRepositories(db *mongodb.DB) *Repositories {
	return &Repositories{
		ChatMode:       mongodb.NewChatModeRepository(db),
		Session:        mongodb.NewChatSessionRepository(db),
		Activity:       mongodb.NewChatActivityRepository(db),
		PurchaseIntent: mongodb.NewPurchaseIntentRepository(db),
	}
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
	whitelistService service.WhitelistService,
) usecase.MessageUsecase {
	return usecase.NewMessageUsecase(
		repos.ChatMode,
		repos.Session,
		repos.Activity,
		chatAPIClient,
		genkitService,
		whitelistService,
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
			return consumer.Stop(ctx)
		},
	})
}

func InitializeDefaultChatModes(lc fx.Lifecycle, initializer service.ChatModeInitializer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := initializer.InitializeDefaultChatModes(ctx); err != nil {
				log := logger.MustNamed("app")
				log.Error("Failed to initialize default chat modes", "error", err)
				// Don't fail the application startup for this optional feature
			}
			return nil
		},
	})
}
