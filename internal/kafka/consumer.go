package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/carousell/ct-go/pkg/logger"
	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/carousell/ct-go/pkg/workerpool"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/pkg/models"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Consumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type kafkaConsumer struct {
	reader         *kafka.Reader
	metrics        *prometheus.HistogramVec
	numWorkers     int
	consumeTimeout time.Duration
	messageHandler MessageHandler
	shutdowner     fx.Shutdowner
	done           chan struct{}
	workerPool     workerpool.Pool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(
	shutdowner fx.Shutdowner,
	cfg *config.KafkaConfig,
	handler MessageHandler,
) (Consumer, error) {
	if !cfg.Enabled {
		return &noopConsumer{}, nil
	}

	metrics, err := util.GetHistogramVec("kafka_messages_consumed", "status", "topic", "group")
	if err != nil {
		return nil, fmt.Errorf("get histogram vec: %w", err)
	}

	readerConfig := kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		StartOffset: kafka.LastOffset,
	}

	numWorkers := 4 // configurable if needed
	wp := workerpool.New(numWorkers)

	return &kafkaConsumer{
		reader:         kafka.NewReader(readerConfig),
		metrics:        metrics,
		numWorkers:     numWorkers,
		consumeTimeout: 30 * time.Second,
		messageHandler: handler,
		done:           make(chan struct{}),
		workerPool:     wp,
	}, nil
}

func (c *kafkaConsumer) Start(ctx context.Context) error {
	log.Infof(ctx, "Starting Kafka consumer v2 for topic: %s", c.reader.Config().Topic)
	defer c.reader.Close()

	if c.numWorkers == 1 {
		return c.startSingleWorker(ctx)
	}
	return c.startMultiWorker(ctx)
}

func (c *kafkaConsumer) Stop(ctx context.Context) error {
	log.Infof(ctx, "Stopping Kafka consumer v2")
	close(c.done)
	c.workerPool.Close()
	c.workerPool.Wait()
	return c.reader.Close()
}

func (c *kafkaConsumer) startSingleWorker(ctx context.Context) error {
	groupID := c.reader.Config().GroupID
	for ctx.Err() == nil {
		select {
		case <-c.done:
			return nil
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			log.Errorw(ctx, "Error fetching message", "error", err)
			continue
		}

		c.processMessage(ctx, msg, groupID)

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Errorw(ctx, "Failed to commit message", "error", err)
		}
	}
	return nil
}

func (c *kafkaConsumer) startMultiWorker(ctx context.Context) error {
	groupID := c.reader.Config().GroupID

	for ctx.Err() == nil {
		select {
		case <-c.done:
			return nil
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			log.Errorw(ctx, "Error reading message", "error", err)
			continue
		}

		// Submit message processing to workerpool
		c.workerPool.Run(func() {
			c.processMessage(ctx, msg, groupID)
		})
	}
	return nil
}

func (c *kafkaConsumer) processMessage(ctx context.Context, msg kafka.Message, groupID string) {
	start := time.Now()
	lagMs := start.Sub(msg.Time).Milliseconds()

	duration, err := c.handle(ctx, msg)

	code := getCode(err)
	content := "success"
	if err != nil {
		content = err.Error()
		log.Errorw(ctx, "Error processing message", "error", err)
	}

	level := getLogLevel(code)
	log.Logw(ctx, level, content,
		"code", code,
		"duration_ms", duration.Milliseconds(),
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"lag_ms", lagMs,
		"key", string(msg.Key),
		"value", json.RawMessage(msg.Value),
	)

	c.metrics.
		WithLabelValues(code.String(), msg.Topic, groupID).
		Observe(duration.Seconds())
}

func (c *kafkaConsumer) handle(msgCtx context.Context, msg kafka.Message) (duration time.Duration, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PANIC RECOVER: %+v", r)
		}
	}()

	start := time.Now()
	defer func() {
		duration = time.Since(start)
	}()

	// Parse Kafka message into the actual format
	var kafkaMessage models.KafkaMessage
	if err := json.Unmarshal(msg.Value, &kafkaMessage); err != nil {
		return 0, fmt.Errorf("failed to unmarshal kafka message: %w", err)
	}

	// Only process message.sent events
	if kafkaMessage.Pattern != "message.sent" {
		log.Infow(msgCtx, "Ignoring non-message.sent event", "pattern", kafkaMessage.Pattern)
		return 0, nil
	}

	// Skip messages from the bot itself to prevent infinite loops
	if kafkaMessage.Data.SenderID == "chat-bot" {
		log.Infow(msgCtx, "Skipping message from bot itself to prevent loops",
			"sender_id", kafkaMessage.Data.SenderID,
			"channel_id", kafkaMessage.Data.ChannelID)
		return 0, nil
	}

	// Convert to internal IncomingMessage format
	incomingMessage := models.IncomingMessage{
		ChannelID: kafkaMessage.Data.ChannelID,
		CreatedAt: kafkaMessage.Data.CreatedAt,
		SenderID:  kafkaMessage.Data.SenderID,
		Message:   kafkaMessage.Data.Message,
		Metadata: models.IncomingMessageMeta{
			LLM: models.LLMMetadata{
				ChatMode: "sales_assistant",
			},
		}, // Initialize with empty metadata for now
	}

	log.Infow(msgCtx, "Processing Kafka message",
		"channel_id", incomingMessage.ChannelID,
		"sender_id", incomingMessage.SenderID)

	ctx, cancel := context.WithTimeout(msgCtx, c.consumeTimeout)
	defer cancel()

	return 0, c.messageHandler.HandleMessage(ctx, &incomingMessage)
}

func getCode(err error) codes.Code {
	if errors.Is(err, context.DeadlineExceeded) {
		return codes.DeadlineExceeded
	}
	if errors.Is(err, context.Canceled) {
		return codes.Canceled
	}
	st, ok := status.FromError(err)
	if !ok {
		return status.Code(errors.Unwrap(err))
	}
	return st.Code()
}

// noopConsumer is used when Kafka is disabled
type noopConsumer struct{}

func (n *noopConsumer) Start(ctx context.Context) error {
	log.Infof(ctx, "Kafka consumer is disabled")
	return nil
}

func (n *noopConsumer) Stop(ctx context.Context) error {
	return nil
}

func getLogLevel(code codes.Code) logger.Level {
	switch code {
	case codes.OK:
		return logger.InfoLevel
	case codes.Canceled,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.Unimplemented,
		codes.OutOfRange:
		return logger.WarnLevel
	default:
		return logger.ErrorLevel
	}
}
