# Service Plugin Architecture Plan

## Overview

Create a partner abstraction layer to support multiple messaging platforms (Chotot chat-api, Facebook Messenger, future platforms) with bidirectional message synchronization and infinite loop prevention.

## Key Clarifications

- **chat-api is Chotot's chat platform** - unified Chotot partner handles both chat and products
- **Remove ItemName/ItemPrice from Channel** - use flexible Metadata instead
- **Replace ExternalChannelID with Partner field** - structure: `{ChannelID: "external_id", Name: "chotot|facebook..."}`

## Architecture Components

### 1. Partner Interface Definition

- **Location**: `internal/repo/partners/partner.go`
- Define `Partner` interface with core messaging operations:
  - `ListMessages(ctx, channelID, limit, beforeTs) ([]Message, error)`
  - `SendMessage(ctx, channelID, senderID, content) error`
  - `GetChannelInfo(ctx, channelID) (*ChannelInfo, error)`
  - `GetUserProducts(ctx, userID, limit, page) ([]Product, error)`
  - `GetPartnerType() string` (returns "chotot", "facebook", etc.)

### 2. Partner Implementations

- **Chotot Partner**: `internal/repo/partners/chotot_partner.go`
  - Unified wrapper around existing `chatapi.Client` (Chotot's chat platform) and `chotot.Client` (products)
  - Implement all Partner interface methods (messaging + products)
- **Future Partners**: Framework ready for Facebook Messenger, Telegram, etc.

### 3. Partner Registry & Detection

- **Location**: `internal/repo/partners/registry.go`
- `PartnerRegistry` to manage partner instances
- `DetectPartner(channelID) Partner` logic based on channel.Partner.Name
- Support for multiple partners per system

### 4. Enhanced Channel Model

```go
type Channel struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Partner    ChannelPartner      `bson:"partner" json:"partner"`
    Name      string             `bson:"name" json:"name"`
    Context   string             `bson:"context" json:"context"`
    Type      string             `bson:"type" json:"type"`
    Metadata  map[string]interface{} `bson:"metadata" json:"metadata"` // ItemName, ItemPrice, etc.
    // ... other fields
}

type ChannelPartner struct {
    ChannelID string `bson:"channel_id" json:"channel_id"` // external partner channel ID
    Name      string `bson:"name" json:"name"`             // "chotot", "facebook", etc.
}
```

### 5. Message Sync Architecture

- **Inbound Flow**: Kafka → PartnerDetection → Channel creation/lookup → Message persistence → Socket broadcast
- **Outbound Flow**: API send → Message persistence → Partner detection → External partner send
- **Loop Prevention**:
  - Track message source (kafka vs api)
  - Implement message deduplication using external message IDs
  - Add sync status tracking per partner

### 6. Enhanced Use Cases

- Update `ChatUseCase.ProcessIncomingMessage` to use partner detection
- Update `ChatUseCase.SendMessage` to support partner-specific sending
- Add partner-aware channel synchronization methods

## Questions/Uncertainties

### Technical Decisions Needed:

1. **Message Deduplication Strategy**:

   - Should we use external message IDs, content hash, or timestamp-based deduplication?
   - How long should we retain deduplication records?
   - **My Recommendation**: Use external message IDs with 24-hour retention window

2. **Partner Priority**: When multiple partners support the same channel, which takes precedence?

   - **My Recommendation**: Use channel.Partner.Name as authoritative source

3. **Error Handling**: How should we handle partial partner failures (e.g., Chotot down)?

   - **My Recommendation**: Continue processing with graceful degradation, log failures

4. **Configuration**: Should partner detection be:
   - Based on channel.Partner.Name (recommended)
   - Fallback pattern-based detection for migration

### Data Model Questions:

1. **Backward Compatibility**: How should we handle existing channels with ExternalChannelID?

   - **My Recommendation**: Migrate ExternalChannelID → Partner{ChannelID: old_external_id, Name: "chotot"}

2. **Metadata Schema**: What should go in flexible Metadata field?

   - ItemName, ItemPrice (from current model)
   - Partner-specific channel settings
   - Custom channel properties

3. **Sync State**: Do we need per-channel, per-partner sync status tracking?
   - **My Recommendation**: Add sync tracking to prevent infinite loops

## Implementation Steps

### Phase 1: Foundation

1. ✅ Define Partner interface and base structures (`internal/repo/partners/partner.go`)
2. Create partner registry and detection logic (`internal/repo/partners/registry.go`)
3. Implement unified Chotot partner wrapper (`internal/repo/partners/chotot_partner.go`)

### Phase 2: Data Model Enhancement

4. Update Channel model: remove ItemName/ItemPrice, replace ExternalChannelID with Partner field
5. Create database migration for existing channels
6. Update repository interfaces to support new channel structure

### Phase 3: Message Flow Integration

7. Update `ChatUseCase.ProcessIncomingMessage` to use partner detection
8. Update `ChatUseCase.SendMessage` to support partner routing
9. Implement message deduplication and loop prevention logic

### Phase 4: Testing & Polish

10. Add comprehensive error handling and logging for partner operations
11. Implement health checks and monitoring for each partner
12. Run tests and validate the implementation
13. Add partner-aware metrics and monitoring

## Current Architecture Analysis

### Existing Message Flow

```
Kafka → ChatUseCase.ProcessIncomingMessage → Chat DB + Socket
API → ChatUseCase.SendMessage → Chat DB + chat-api (async)
```

### Proposed Message Flow

```
Kafka → PartnerDetection → ChatUseCase.ProcessIncomingMessage → Chat DB + Socket
API → ChatUseCase.SendMessage → Chat DB + PartnerRouting → External Partner (async)
```

### Migration Strategy

1. **Channel Model Migration**:

   ```sql
   // Migrate existing channels
   db.channels.updateMany(
     { partner: { $exists: false } },
     {
       $set: {
         partner: {
           channel_id: "$external_channel_id",
           name: "chotot"
         }
       },
       $unset: {
         external_channel_id: "",
         item_name: "",
         item_price: ""
       }
     }
   )
   ```

2. **Gradual Partner Integration**:
   - Deploy partner abstraction layer
   - Migrate Chotot integration first
   - Add new partner types incrementally
   - Maintain backward compatibility

## Loop Prevention Strategy

1. **Source Tracking**: Mark messages with source (kafka, api, partner)
2. **Deduplication**: Use external message IDs to detect duplicates
3. **Sync Windows**: Implement time-based sync windows to prevent rapid loops
4. **Circuit Breaker**: Stop processing if loop detected

## Testing Strategy

- Unit tests for unified Chotot partner implementation
- Integration tests for bidirectional sync scenarios
- Load tests for loop prevention mechanisms
- Channel migration testing
- Backward compatibility testing

This plan provides a cleaner, unified approach with Chotot as the primary partner and a flexible channel model ready for future platforms.
