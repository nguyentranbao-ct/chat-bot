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
				log.Debugw(ctx, "Ignoring non-message.sent event", "pattern", kafkaMessage.Pattern)
				return nil
			}

			log.Debugw(ctx, "Processing Kafka message",
				"channel_id", kafkaMessage.Data.ChannelID,
				"sender_id", kafkaMessage.Data.SenderID)

			// Transform Kafka message to our internal format with partner detection
			data := kafkaMessage.Data

			// Use ChatUseCase.ProcessIncomingMessage for message storage and socket events
			// Skip LLM processing for Kafka messages
			return chatUsecase.ProcessIncomingMessage(ctx, data)
		},
	})
}
