package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
)

func StartConsumeMessages(
	sd fx.Shutdowner,
	lc fx.Lifecycle,
	conf *config.Config,
	messageUsecase usecase.MessageUsecase,
	chatUsecase *usecase.ChatUseCase,
) error {
	if !conf.Kafka.Enabled {
		log.Warnf(context.Background(), "Kafka consumer is disabled in configuration")
		return nil
	}
	return startKafkaConsumer(consumerOptions{
		sd: sd,
		lc: lc,
		readerConf: kafka.ReaderConfig{
			Brokers:     conf.Kafka.Brokers,
			GroupID:     conf.Kafka.GroupID,
			GroupTopics: []string{conf.Kafka.Topic},
		},
		maxWorkers:     5,
		consumeTimeout: 30 * 1e9, // 30 seconds
		handler: func(ctx context.Context, msg kafka.Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := make([]byte, 4096)
					length := runtime.Stack(stack, false)
					err = fmt.Errorf("PANIC RECOVER: %+v / %s", r, string(stack[:length]))
				}
			}()

			// Parse Kafka message into the actual format
			var kafkaMessage models.KafkaMessage
			if err := json.Unmarshal(msg.Value, &kafkaMessage); err != nil {
				return fmt.Errorf("failed to unmarshal kafka message: %w", err)
			}

			// Only process message.sent events
			if kafkaMessage.Pattern != "message.sent" {
				log.Infow(ctx, "Ignoring non-message.sent event", "pattern", kafkaMessage.Pattern)
				return nil
			}

			log.Infow(ctx, "Processing Kafka message",
				"channel_id", kafkaMessage.Data.ChannelID,
				"sender_id", kafkaMessage.Data.SenderID)

			// First, sync the message to our chat database
			if err := chatUsecase.ProcessIncomingMessage(ctx, kafkaMessage.Data); err != nil {
				log.Errorw(ctx, "Failed to sync message to chat database", "error", err)
				// Continue processing for LLM even if chat sync fails
			}

			// Then process with LLM if needed (existing logic)
			incomingMessage := models.IncomingMessage{
				ChannelID: kafkaMessage.Data.ChannelID,
				CreatedAt: kafkaMessage.Data.CreatedAt,
				SenderID:  kafkaMessage.Data.SenderID,
				Message:   kafkaMessage.Data.Message,
				Metadata: models.IncomingMessageMeta{
					LLM: models.LLMMetadata{
						ChatMode: "sales_assistant",
					},
				},
			}

			return messageUsecase.ProcessMessage(ctx, incomingMessage)
		},
	})
}
