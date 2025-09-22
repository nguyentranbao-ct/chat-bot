# Service Plugin Architecture Plan

## Overview

Create a vendor abstraction layer to support multiple messaging platforms (Chotot chat-api, Facebook Messenger, future platforms) with bidirectional message synchronization and infinite loop prevention.

## Key Clarifications

- **chat-api is Chotot's chat platform** - unified Chotot vendor handles both chat and products
- **Remove ItemName/ItemPrice from Channel** - use flexible Metadata instead
- **Replace ExternalChannelID with Vendor field** - structure: `{ChannelID: "external_id", Name: "chotot|facebook..."}`

## Architecture Components

### 1. Vendor Interface Definition

- **Location**: `internal/repo/vendors/vendor.go`
- Define `Vendor` interface with core messaging operations:
  - `ListMessages(ctx, channelID, limit, beforeTs) ([]Message, error)`
  - `SendMessage(ctx, channelID, senderID, content) error`
  - `GetChannelInfo(ctx, channelID) (*ChannelInfo, error)`
  - `GetUserProducts(ctx, userID, limit, page) ([]Product, error)`
  - `GetVendorType() string` (returns "chotot", "facebook", etc.)

### 2. Vendor Implementations

- **Chotot Vendor**: `internal/repo/vendors/chotot_vendor.go`
  - Unified wrapper around existing `chatapi.Client` (Chotot's chat platform) and `chotot.Client` (products)
  - Implement all Vendor interface methods (messaging + products)
- **Future Vendors**: Framework ready for Facebook Messenger, Telegram, etc.

### 3. Vendor Registry & Detection

- **Location**: `internal/repo/vendors/registry.go`
- `VendorRegistry` to manage vendor instances
- `DetectVendor(channelID) Vendor` logic based on channel.Vendor.Name
- Support for multiple vendors per system

### 4. Enhanced Channel Model

```go
type Channel struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Vendor    ChannelVendor      `bson:"vendor" json:"vendor"`
    Name      string             `bson:"name" json:"name"`
    Context   string             `bson:"context" json:"context"`
    Type      string             `bson:"type" json:"type"`
    Metadata  map[string]interface{} `bson:"metadata" json:"metadata"` // ItemName, ItemPrice, etc.
    // ... other fields
}

type ChannelVendor struct {
    ChannelID string `bson:"channel_id" json:"channel_id"` // external vendor channel ID
    Name      string `bson:"name" json:"name"`             // "chotot", "facebook", etc.
}
```

### 5. Message Sync Architecture

- **Inbound Flow**: Kafka → VendorDetection → Channel creation/lookup → Message persistence → Socket broadcast
- **Outbound Flow**: API send → Message persistence → Vendor detection → External vendor send
- **Loop Prevention**:
  - Track message source (kafka vs api)
  - Implement message deduplication using external message IDs
  - Add sync status tracking per vendor

### 6. Enhanced Use Cases

- Update `ChatUseCase.ProcessIncomingMessage` to use vendor detection
- Update `ChatUseCase.SendMessage` to support vendor-specific sending
- Add vendor-aware channel synchronization methods

## Questions/Uncertainties

### Technical Decisions Needed:

1. **Message Deduplication Strategy**:

   - Should we use external message IDs, content hash, or timestamp-based deduplication?
   - How long should we retain deduplication records?
   - **My Recommendation**: Use external message IDs with 24-hour retention window

2. **Vendor Priority**: When multiple vendors support the same channel, which takes precedence?

   - **My Recommendation**: Use channel.Vendor.Name as authoritative source

3. **Error Handling**: How should we handle partial vendor failures (e.g., Chotot down)?

   - **My Recommendation**: Continue processing with graceful degradation, log failures

4. **Configuration**: Should vendor detection be:
   - Based on channel.Vendor.Name (recommended)
   - Fallback pattern-based detection for migration

### Data Model Questions:

1. **Backward Compatibility**: How should we handle existing channels with ExternalChannelID?

   - **My Recommendation**: Migrate ExternalChannelID → Vendor{ChannelID: old_external_id, Name: "chotot"}

2. **Metadata Schema**: What should go in flexible Metadata field?

   - ItemName, ItemPrice (from current model)
   - Vendor-specific channel settings
   - Custom channel properties

3. **Sync State**: Do we need per-channel, per-vendor sync status tracking?
   - **My Recommendation**: Add sync tracking to prevent infinite loops

## Implementation Steps

### Phase 1: Foundation

1. ✅ Define Vendor interface and base structures (`internal/repo/vendors/vendor.go`)
2. Create vendor registry and detection logic (`internal/repo/vendors/registry.go`)
3. Implement unified Chotot vendor wrapper (`internal/repo/vendors/chotot_vendor.go`)

### Phase 2: Data Model Enhancement

4. Update Channel model: remove ItemName/ItemPrice, replace ExternalChannelID with Vendor field
5. Create database migration for existing channels
6. Update repository interfaces to support new channel structure

### Phase 3: Message Flow Integration

7. Update `ChatUseCase.ProcessIncomingMessage` to use vendor detection
8. Update `ChatUseCase.SendMessage` to support vendor routing
9. Implement message deduplication and loop prevention logic

### Phase 4: Testing & Polish

10. Add comprehensive error handling and logging for vendor operations
11. Implement health checks and monitoring for each vendor
12. Run tests and validate the implementation
13. Add vendor-aware metrics and monitoring

## Current Architecture Analysis

### Existing Message Flow

```
Kafka → ChatUseCase.ProcessIncomingMessage → Chat DB + Socket
API → ChatUseCase.SendMessage → Chat DB + chat-api (async)
```

### Proposed Message Flow

```
Kafka → VendorDetection → ChatUseCase.ProcessIncomingMessage → Chat DB + Socket
API → ChatUseCase.SendMessage → Chat DB + VendorRouting → External Vendor (async)
```

### Migration Strategy

1. **Channel Model Migration**:

   ```sql
   // Migrate existing channels
   db.channels.updateMany(
     { vendor: { $exists: false } },
     {
       $set: {
         vendor: {
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

2. **Gradual Vendor Integration**:
   - Deploy vendor abstraction layer
   - Migrate Chotot integration first
   - Add new vendor types incrementally
   - Maintain backward compatibility

## Loop Prevention Strategy

1. **Source Tracking**: Mark messages with source (kafka, api, vendor)
2. **Deduplication**: Use external message IDs to detect duplicates
3. **Sync Windows**: Implement time-based sync windows to prevent rapid loops
4. **Circuit Breaker**: Stop processing if loop detected

## Testing Strategy

- Unit tests for unified Chotot vendor implementation
- Integration tests for bidirectional sync scenarios
- Load tests for loop prevention mechanisms
- Channel migration testing
- Backward compatibility testing

This plan provides a cleaner, unified approach with Chotot as the primary vendor and a flexible channel model ready for future platforms.
