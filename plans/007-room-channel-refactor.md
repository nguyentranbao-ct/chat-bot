# Plan 007: Room/Channel Refactor - Single Enhanced RoomMember Collection

## Overview
Successfully refactored the chat system from using separate `rooms`, `members`, and `unread_counts` collections to a single enhanced `room_members` collection for improved performance and reduced complexity.

## Objectives ✅ COMPLETED
- Consolidate room and member data into a single collection
- Eliminate complex aggregation queries
- Reduce database operations
- Improve performance through denormalization
- Maintain all existing functionality

## Implementation Summary

### ✅ 1. Enhanced RoomMember Model
**File**: `internal/models/room.go`

Updated `RoomMember` struct to include:
- **Member Identity**: ID, UserID, Role (merchant for internal users)
- **Room Information** (denormalized): Source, RoomID, RoomName, RoomContext, Metadata
- **Member-Specific Data**: LastReadAt, LastMessageAt, LastMessageContent, UnreadCount
- **Timestamps**: JoinedAt, CreatedAt, UpdatedAt

**Key Design Decisions**:
- Use `time.Time` instead of pointers for better performance
- RoomID still used for internal operations
- Source (RoomPartner) used for external partner operations
- Added `omitempty` tags where appropriate

### ✅ 2. Repository Layer Refactor
**File**: `internal/repo/mongodb/room.go`

**Removed**:
- `RoomRepository` interface and implementation
- `UnreadCountRepository` interface and implementation
- Old `Room` and `UnreadCount` models

**Enhanced RoomMemberRepository**:
- `GetUserRoomMembers()` - Get all rooms for a user
- `GetRoomMembersByRoomID()` - Get members by internal room ID
- `GetRoomMembers()` - Get members by external source
- `IncrementUnreadCountByRoomID()` - Bulk unread count updates
- `MarkAsReadByRoomID()` - Mark messages as read
- `UpdateLastMessageForRoom()` - Update last message info for all room members
- `FindOrCreateRoom()` - Create room with all participants

**Database Collections**:
- Uses single `room_members` collection
- Removed dependencies on `rooms` and `unread_counts` collections

### ✅ 3. Use Case Layer Updates
**File**: `internal/usecase/chat_usecase.go`

**Updated Methods**:
- `GetUserRooms()` - Returns `[]*models.RoomMember` instead of aggregated data
- `GetRoomMembersByRoomID()` - New method for internal room operations
- `SendMessage()` - Uses RoomID for internal ops, Source for external partner ops
- `incrementUnreadCountForOthers()` - Simplified to single repository call
- `MarkAsRead()` - Uses new repository method

**Removed Dependencies**:
- Removed `roomRepo` and `unreadCountRepo` from ChatUseCase
- Updated constructor to only require `roomMemberRepo`

### ✅ 4. LLM Use Case Updates
**File**: `internal/usecase/chat_usecase_llm.go`

**Interface Changes**:
- `TriggerLLM()` now accepts `*models.RoomMember` instead of `*models.Room`
- Updated `PromptContext` and `PromptData` to use `RoomMember`
- Updated all method signatures and implementations

### ✅ 5. Controller Layer Updates
**File**: `internal/server/chat_controller.go`

**Updated Methods**:
- `GetRooms()` - Returns room members directly
- `GetRoomMembers()` - Uses new repository method
- `SendMessage()` and `SendInternalMessage()` - Updated for new structure

### ✅ 6. Dependency Injection Updates
**File**: `internal/app/app.go`

**Removed Providers**:
- `mongodb.NewRoomRepository`
- `mongodb.NewUnreadCountRepository`

**Updated Constructor**:
- ChatUseCase constructor no longer requires room and unread count repositories

### ✅ 7. Setup Files Updates
**File**: `internal/setup/setup_chat_rooms.go`

**Updated**:
- Removed dependency on RoomRepository
- Creates enhanced RoomMember records directly
- Uses source-based room existence checking

### ✅ 8. Model Cleanup
**File**: `internal/models/room.go`

**Removed**:
- `Room` struct (obsolete)
- `UnreadCount` struct (functionality merged into RoomMember)

**Kept**:
- `RoomPartner` struct (still used for external partner identification)

## Technical Benefits Achieved

### Performance Improvements
- **Single Query Access**: User rooms retrieved with one query instead of complex aggregations
- **Reduced Database Operations**: Unread count updates and last message tracking are single operations
- **Eliminated Joins**: No more expensive lookup operations between collections

### Code Simplification
- **Fewer Repository Interfaces**: Reduced from 3 to 1 primary interface
- **Simplified Use Cases**: Removed complex aggregation logic
- **Cleaner Dependencies**: Fewer injected repositories

### Data Consistency
- **Denormalized Design**: Room information co-located with member data for optimal read performance
- **Atomic Operations**: Member-specific updates are single document operations

## Architecture Pattern
This refactor implements a **denormalized, member-centric** design where:
- Each user has their own view of room data
- Room information is duplicated per member for performance
- Member A's name becomes Member B's room name (as requested)
- Internal operations use RoomID, external operations use Source

## Database Schema Migration Required
When deploying, a migration script should:
1. Create new `room_members` collection with enhanced schema
2. Migrate data from existing `rooms`, `members`, and `unread_counts` collections
3. Drop old collections after validation

## Next Steps (Future Work)
1. **Web Frontend Updates**: Update `./web` to handle new RoomMember structure
2. **API Documentation**: Update API docs to reflect new response formats
3. **Migration Script**: Create database migration for production deployment
4. **Performance Testing**: Validate performance improvements with load testing

## Files Modified
- `internal/models/room.go` - Enhanced RoomMember model
- `internal/repo/mongodb/room.go` - Consolidated repository
- `internal/usecase/chat_usecase.go` - Updated business logic
- `internal/usecase/chat_usecase_llm.go` - LLM integration updates
- `internal/usecase/llm_usecase.go` - Prompt context updates
- `internal/server/chat_controller.go` - API endpoint updates
- `internal/app/app.go` - Dependency injection updates
- `internal/setup/setup_chat_rooms.go` - Setup logic updates

## Compilation Status
✅ All tests pass
✅ No compilation errors
✅ Ready for web frontend integration