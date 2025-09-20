package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/IBM/sarama"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

type consumer struct {
	config           *config.KafkaConfig
	consumerGroup    sarama.ConsumerGroup
	messageHandler   MessageHandler
	whitelistService WhitelistService
	ready            chan bool
	done             chan struct{}
	wg               sync.WaitGroup
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.KafkaConfig, handler MessageHandler, whitelist WhitelistService) (Consumer, error) {
	if !cfg.Enabled {
		return &noopConsumer{}, nil
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Group.Session.Timeout = 10000
	saramaConfig.Consumer.Group.Heartbeat.Interval = 3000

	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &consumer{
		config:           cfg,
		consumerGroup:    consumerGroup,
		messageHandler:   handler,
		whitelistService: whitelist,
		ready:            make(chan bool),
		done:             make(chan struct{}),
	}, nil
}

// Start begins consuming messages from Kafka
func (c *consumer) Start(ctx context.Context) error {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.done:
				return
			default:
				if err := c.consumerGroup.Consume(ctx, []string{c.config.Topic}, c); err != nil {
					log.Printf("Error consuming from Kafka: %v", err)
				}
			}
		}
	}()

	// Wait for consumer to be ready
	<-c.ready
	log.Printf("Kafka consumer started successfully, consuming from topic: %s", c.config.Topic)
	return nil
}

// Stop gracefully stops the consumer
func (c *consumer) Stop() error {
	close(c.done)
	if err := c.consumerGroup.Close(); err != nil {
		log.Printf("Error closing consumer group: %v", err)
		return err
	}
	c.wg.Wait()
	log.Println("Kafka consumer stopped")
	return nil
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *consumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (c *consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			if err := c.processMessage(session.Context(), message); err != nil {
				log.Printf("Error processing message: %v", err)
				// Continue processing other messages even if one fails
				continue
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// processMessage handles individual Kafka messages
func (c *consumer) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var incomingMessage models.IncomingMessage
	if err := json.Unmarshal(message.Value, &incomingMessage); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Check if channel is whitelisted
	if !c.whitelistService.IsChannelAllowed(incomingMessage.ChannelID) {
		log.Printf("Ignoring message from non-whitelisted channel: %s", incomingMessage.ChannelID)
		return nil
	}

	log.Printf("Processing Kafka message from channel: %s, sender: %s",
		incomingMessage.ChannelID, incomingMessage.SenderID)

	return c.messageHandler.HandleMessage(ctx, &incomingMessage)
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
