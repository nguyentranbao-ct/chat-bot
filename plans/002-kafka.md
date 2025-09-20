# Plan 002: Kafka Integration for Message Events

## Overview

Implement Kafka consumer to process message events from the `chat.event.messages` topic as an alternative trigger mechanism to the existing HTTP API endpoint.

## Requirements

- Consume messages from Kafka topic: `chat.event.messages`
- Kafka brokers: `kafka-08.ct.dev:9200`
- Channel whitelist configuration to control which channels trigger processing
- Maintain existing HTTP API functionality
- Integrate with current message processing pipeline

## Implementation Plan

### 1. Dependencies and Configuration

- Add Kafka client library to `go.mod` (e.g., `github.com/IBM/sarama` or `github.com/confluentinc/confluent-kafka-go`)
- Extend `internal/config` to include Kafka configuration:
  - Broker addresses
  - Topic name
  - Consumer group ID
  - Channel whitelist settings

### 2. Kafka Consumer Implementation

**File: `internal/kafka/consumer.go`**
- Implement Kafka consumer with proper error handling
- Message deserialization from Kafka to `models.IncomingMessage`
- Consumer group management for scalability
- Graceful shutdown handling

**File: `internal/kafka/interfaces.go`**
- Define interfaces for testability and abstraction

### 3. Channel Whitelist Service

**File: `internal/service/whitelist.go`**
- Implement channel whitelist checking
- Configuration-based or database-backed whitelist
- Interface for easy testing and future extensions

### 4. Integration with Existing Pipeline

**Modify: `internal/app/app.go`**
- Add Kafka consumer to FX dependency injection
- Wire whitelist service
- Start Kafka consumer alongside HTTP server

**Modify: `internal/usecase/message_usecase.go`**
- Ensure message processing logic works for both HTTP and Kafka sources
- Add source tracking for observability

### 5. Configuration Updates

**Environment Variables:**
```
KAFKA_BROKERS=kafka-08.ct.dev:9200
KAFKA_TOPIC=chat.event.messages
KAFKA_CONSUMER_GROUP=chat-bot-consumers
KAFKA_CHANNEL_WHITELIST=channel1,channel2,channel3
```

### 6. Error Handling and Monitoring

- Implement proper error handling for Kafka connectivity issues
- Add metrics for message processing rates
- Log Kafka consumer health and message processing status
- Dead letter queue consideration for failed messages

### 7. Testing Strategy

- Unit tests for Kafka consumer logic
- Integration tests with test Kafka cluster
- Mock Kafka consumer for usecase testing
- End-to-end tests with both HTTP and Kafka message sources

## Files to Create/Modify

### New Files:
- `internal/kafka/consumer.go`
- `internal/kafka/interfaces.go`
- `internal/service/whitelist.go`
- `internal/service/interfaces.go`

### Modified Files:
- `internal/config/config.go` - Add Kafka configuration
- `internal/app/app.go` - Wire Kafka consumer
- `internal/usecase/message_usecase.go` - Support multiple message sources
- `go.mod` - Add Kafka dependencies
- `Makefile` - Add Kafka-related development commands

## Deployment Considerations

- Ensure Kafka consumer starts after MongoDB connection is established
- Configure consumer group for horizontal scaling
- Monitor consumer lag and processing throughput
- Consider graceful shutdown order (HTTP server, then Kafka consumer)

## Future Enhancements

- Dynamic channel whitelist updates without restart
- Message filtering at Kafka level using headers
- Support for multiple topics
- Dead letter queue implementation
- Kafka message retry mechanisms

## Success Criteria

- [ ] Kafka consumer successfully connects to brokers
- [ ] Messages from whitelisted channels are processed
- [ ] Non-whitelisted channels are ignored
- [ ] Existing HTTP API continues to work unchanged
- [ ] Proper error handling and logging implemented
- [ ] Unit and integration tests pass
- [ ] Performance metrics available for monitoring