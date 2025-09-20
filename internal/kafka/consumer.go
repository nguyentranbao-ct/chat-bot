package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type kafkaConsumer struct {
	reader           *kafka.Reader
	metrics          *prometheus.HistogramVec
	numWorkers       int
	consumeTimeout   time.Duration
	messageHandler   MessageHandler
	whitelistService WhitelistService
	shutdowner       fx.Shutdowner
	done             chan struct{}
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(
	shutdowner fx.Shutdowner,
	cfg *config.KafkaConfig,
	handler MessageHandler,
	whitelist WhitelistService,
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

	return &kafkaConsumer{
		reader:           kafka.NewReader(readerConfig),
		metrics:          metrics,
		numWorkers:       4, // configurable if needed
		consumeTimeout:   30 * time.Second,
		messageHandler:   handler,
		whitelistService: whitelist,
		done:             make(chan struct{}),
	}, nil
}

func (c *kafkaConsumer) Start(ctx context.Context) error {
	log.Printf("Starting Kafka consumer v2 for topic: %s", c.reader.Config().Topic)
	defer c.reader.Close()

	if c.numWorkers == 1 {
		return c.startSingleWorker(ctx)
	}
	return c.startMultiWorker(ctx)
}

func (c *kafkaConsumer) Stop() error {
	log.Println("Stopping Kafka consumer v2")
	close(c.done)
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
			log.Printf("Error fetching message: %v", err)
			continue
		}

		c.processMessage(ctx, msg, groupID)

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Failed to commit message: %v", err)
		}
	}
	return nil
}

func (c *kafkaConsumer) startMultiWorker(ctx context.Context) error {
	// For now, use simple goroutines instead of workerpool
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
			log.Printf("Error reading message: %v", err)
			continue
		}

		go c.processMessage(ctx, msg, groupID)
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
		log.Printf("Error processing message: %v", err)
	}

	log.Printf("Processed message - topic: %s, partition: %d, offset: %d, duration: %vms, lag: %vms, status: %s",
		msg.Topic, msg.Partition, msg.Offset, duration.Milliseconds(), lagMs, content)

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

	// Parse Kafka message into our expected format
	var incomingMessage models.IncomingMessage
	if err := json.Unmarshal(msg.Value, &incomingMessage); err != nil {
		return 0, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Check if channel is whitelisted
	if !c.whitelistService.IsChannelAllowed(incomingMessage.ChannelID) {
		log.Printf("Ignoring message from non-whitelisted channel: %s", incomingMessage.ChannelID)
		return 0, nil
	}

	log.Printf("Processing Kafka message from channel: %s, sender: %s",
		incomingMessage.ChannelID, incomingMessage.SenderID)

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
	log.Println("Kafka consumer is disabled")
	return nil
}

func (n *noopConsumer) Stop() error {
	return nil
}
