# User Account Linking System Implementation Plan

## Overview

Implement a simple user account linking system that allows users to link external service accounts (starting with Chotot) to enable personalized chat experiences and product listing functionality.

## 1. Data Models Design

### User Model (`pkg/models/user.go`)

```go
type User struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name      string             `bson:"name" json:"name"`
    Email     string             `bson:"email" json:"email"`
    CreatedAt time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
```

**Questions/Uncertainties:**

- Should we add any validation rules for email format? okay, use go validate tag
- Do we need additional user fields like phone number, avatar, status? not yet
- Should we enforce unique email constraint at DB level? yes

### User Attribute Model (`pkg/models/user_attribute.go`)

```go
type UserAttribute struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
    Key       string             `bson:"key" json:"key"`
    Value     string             `bson:"value" json:"value"`
    Tags      []string           `bson:"tags" json:"tags"`
    CreatedAt time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
```

**Examples:**

- Chotot integration: `user_id: ObjectID, key: "chotot_id", value: "123456", tags: ["chotot", "link_id"]`
- Facebook link: `user_id: ObjectID, key: "facebook_id", value: "fb_123", tags: ["facebook", "social"]`

**Questions/Uncertainties:**

- Should we enforce unique constraint on (UserID, Key) to prevent duplicate keys per user? yes
- Should we add validation for specific key formats? not initially
- Should tags be case-sensitive? yes, keep as provided
- Do we need indexing on tags for queries? yes, but into migration files in migrateions/YYMMDD-title.mongodb.js

## 2. Repository Layer

### User Repository (`internal/repo/mongodb/user.go`)

```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id primitive.ObjectID) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id primitive.ObjectID) error
    List(ctx context.Context, limit, offset int) ([]*User, error)
}
```

### User Attribute Repository (`internal/repo/mongodb/user_attribute.go`)

```go
type UserAttributeRepository interface {
    Create(ctx context.Context, attr *UserAttribute) error
    GetByID(ctx context.Context, id primitive.ObjectID) (*UserAttribute, error)
    GetByUserID(ctx context.Context, userID primitive.ObjectID) ([]*UserAttribute, error)
    GetByUserIDAndKey(ctx context.Context, userID primitive.ObjectID, key string) (*UserAttribute, error)
    GetByKey(ctx context.Context, key string) ([]*UserAttribute, error)
    GetByTags(ctx context.Context, tags []string) ([]*UserAttribute, error)
    GetByUserIDAndTags(ctx context.Context, userID primitive.ObjectID, tags []string) ([]*UserAttribute, error)
    Update(ctx context.Context, attr *UserAttribute) error
    Upsert(ctx context.Context, attr *UserAttribute) error
    Delete(ctx context.Context, id primitive.ObjectID) error
    DeleteByUserIDAndKey(ctx context.Context, userID primitive.ObjectID, key string) error
}
```

**Questions/Uncertainties:**

- Should we add pagination support to GetByUserID? no need, we expect few attributes per user
- Do we need batch operations for better performance? not initially
- Should we add soft delete functionality? no need
- Should GetByTags use AND or OR logic for multiple tags? OR logic for flexibility

## 3. Usecase Layer

### User Usecase (`internal/usecase/user_usecase.go`)

```go
type UserUsecase interface {
    CreateUser(ctx context.Context, name, email string) (*User, error)
    GetUser(ctx context.Context, id primitive.ObjectID) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    UpdateUser(ctx context.Context, user *User) error
    DeleteUser(ctx context.Context, id primitive.ObjectID) error

    SetUserAttribute(ctx context.Context, userID primitive.ObjectID, key, value string, tags []string) error
    GetUserAttributes(ctx context.Context, userID primitive.ObjectID) ([]*UserAttribute, error)
    GetUserAttributeByKey(ctx context.Context, userID primitive.ObjectID, key string) (*UserAttribute, error)
    GetUsersByTag(ctx context.Context, tags []string) ([]*User, error)
    GetUserByChototID(ctx context.Context, chototID string) (*User, error)
    RemoveUserAttribute(ctx context.Context, userID primitive.ObjectID, key string) error
}
```

**Questions/Uncertainties:**

- Should SetUserAttribute update existing attributes or create new ones? update existing (upsert behavior)
- Do we need validation for specific key formats? currently no but can add later
- Should GetUsersByTag return users who have any of the tags or all tags? any of the tags (OR logic)

## 4. API Endpoints

### User Management Endpoints

- `POST /api/v1/users` - Create user
  - Body: `{"name": "string", "email": "string"}`
- `GET /api/v1/users/{id}` - Get user by ID
- `PUT /api/v1/users/{id}` - Update user
  - Body: `{"name": "string", "email": "string"}`
- `DELETE /api/v1/users/{id}` - Delete user

### User Attributes Endpoints

- `POST /api/v1/users/{id}/attributes` - Set user attribute
  - Body: `{"key": "string", "value": "string", "tags": ["string"]}`
- `GET /api/v1/users/{id}/attributes` - Get user's attributes
- `GET /api/v1/users/{id}/attributes/{key}` - Get specific attribute by key
- `DELETE /api/v1/users/{id}/attributes/{key}` - Remove attribute by key

**Implementation Notes:**

- No authentication required for endpoints initially
- Basic validation middleware for request bodies
- Self-registration allowed for users
- No rate limiting initially

## 5. Default Data Files

### `internal/usecase/default_users.yaml`

```yaml
---
- name: 'Test User'
  email: 'test@example.com'
- name: 'Demo User'
  email: 'demo@chotot.vn'
```

### `internal/usecase/default_user_attributes.yaml`

```yaml
---
- user_email: 'test@example.com'
  key: 'chotot_id'
  value: '123456'
  tags: ['chotot', 'link_id']
- user_email: 'demo@chotot.vn'
  key: 'chotot_id'
  value: '789012'
  tags: ['chotot', 'link_id']
- user_email: 'test@example.com'
  key: 'facebook_id'
  value: 'fb_test_123'
  tags: ['facebook', 'social']
```

**Implementation Notes:**

- Create initializer script similar to chat_mode_initializer.go
- Add configuration flag to enable/disable default data loading
- Move YAML files to `assets/` directory, keep initializer in usecase package
- Load default data on every startup
- Use upsert behavior for conflicts with existing data

## 6. Chat Mode Modifications

### Update ChatMode Model

Add optional UserID to support user-specific chat modes:

```go
type ChatMode struct {
    // ... existing fields
    UserID *primitive.ObjectID `bson:"user_id,omitempty" json:"user_id,omitempty" yaml:"user_id,omitempty"`
}
```

### Update Chat Mode Selection Logic

Modify message processing to:

1. First look for user-specific chat modes (where UserID matches)
2. Fall back to default chat modes (where UserID is nil)

**Implementation Notes:**

- Users can customize system prompt and tools only initially
- No inheritance between default and user-specific modes
- User-specific modes are unique by (user_id, mode_name) combination
- No versioning for user customizations initially

## 7. Chotot Integration & Product Listing Tool

### New Tool: `ListProducts` (`internal/repo/tools/list_products/`)

```go
type ListProductsInput struct {
    UserID string `json:"user_id"`
    Limit  int    `json:"limit,omitempty"`
    Page   int    `json:"page,omitempty"`
}

type ListProductsOutput struct {
    Products []Product `json:"products"`
    Total    int       `json:"total"`
}

// Our unified Product model
type Product struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Category    string   `json:"category"`
    Price       int      `json:"price"`
    PriceString string   `json:"price_string"`
    Images      []string `json:"images"`
    Source      string   `json:"source"` // e.g., "chotot://1823180"
}
```

### Product Service Interface (Pluggable Design)

```go
type ProductService interface {
    ListUserProducts(ctx context.Context, userID string, limit, page int) ([]Product, int, error)
}

type ProductServiceRegistry interface {
    RegisterService(linkType string, service ProductService)
    GetService(linkType string) (ProductService, bool)
}
```

### Chotot Product Service Implementation

```go
// Internal Chotot models (not exposed in tool output)
type ChototAd struct {
    AdID         int                 `json:"ad_id"`
    ListID       int                 `json:"list_id"`
    AccountID    int                 `json:"account_id"`
    AccountOID   string              `json:"account_oid"`
    Subject      string              `json:"subject"`
    Title        string              `json:"title"`
    Category     int                 `json:"category"`
    BigCate      int                 `json:"bigCate"`
    Price        int                 `json:"price"`
    PriceString  string              `json:"price_string"`
    Region       int                 `json:"region"`
    RegionName   string              `json:"region_name"`
    AreaV2       string              `json:"area_v2"`
    AreaName     string              `json:"area_name"`
    Date         string              `json:"date"`
    Images       []string            `json:"images"`
    Params       []ChototAdParam     `json:"params"`
}

type ChototAdParam struct {
    ID    string `json:"id"`
    Value string `json:"value"`
}

// Mapper function
func (c *ChototProductService) mapToProduct(ad ChototAd) Product {
    return Product{
        ID:          fmt.Sprintf("%d", ad.ListID),
        Name:        ad.Subject,
        Category:    getCategoryName(ad.Category), // map category ID to string
        Price:       ad.Price,
        PriceString: ad.PriceString,
        Images:      ad.Images,
        Source:      fmt.Sprintf("chotot://%d", ad.ListID),
    }
}
```

### Chotot API Integration

**API Endpoint:**

- URL: `https://gateway.chotot.org/v1/public/theia/{CHOTOT_ACCOUNT_OID}`
- Parameters: `limit` (default: 9), `page` (default: 1)
- Method: GET
- Authentication: None required (public endpoint)

**Implementation:**

- Create Chotot API client in `internal/repo/chotot/`
- Implement `ChototProductService` that implements `ProductService` interface
- Tool queries user attributes to find `chotot_id` (account OID)
- Maps Chotot ads to our unified Product model
- Registers service with ProductServiceRegistry as "chotot"

**Product Service Flow:**

1. Tool receives user_id
2. Query user attributes for services with "link_id" tag
3. For each service, get corresponding ProductService from registry
4. Aggregate products from all services
5. Return unified Product list

**Implementation Notes:**

- Fetch fresh product data (no caching initially)
- Handle API rate limiting gracefully with exponential backoff
- Implement basic retry logic for failed requests
- Log API errors and return empty list on failures
- Merge products from multiple services into single unified list
- Tool operates as standalone on-demand tool

## 8. Implementation Dependencies

### Update FX Providers (`internal/app/providers.go`)

Add providers for:

- UserRepository
- UserAttributeRepository
- UserUsecase
- ProductServiceRegistry
- ChototProductService (registered as "chotot")
- User-related HTTP handlers

### Database Collections

- `users` - User documents
- `user_attributes` - User attribute documents

### Required Indexes

- `users.email` (unique)
- `user_attributes.user_id`
- `user_attributes.user_id + user_attributes.key` (unique)
- `user_attributes.tags` (for tag-based queries)
- `chat_modes.user_id` (sparse index)

**Implementation Notes:**

- Create database indexes using migration files in `migrations/YYMMDD-title.mongodb.js`
- No additional performance indexes initially
- No compound indexes for complex queries initially

## 9. Error Handling & Validation

### Input Validation

- Email format validation
- Required fields validation
- External ID format validation per link type

### Error Cases

- Duplicate email registration
- Duplicate user attribute keys for the same user
- Invalid key/value formats
- User not found scenarios
- Attribute not found scenarios

**Implementation Notes:**

- Use basic custom validation for input
- Add validation for attribute keys (alpha-numeric with underscores)
- No caching of external data initially
- Handle partial failures in batch operations not applicable initially

## 10. Testing Strategy

### Unit Tests

- Repository layer tests with mocked MongoDB
- Usecase layer tests with mocked repositories
- Handler tests with mocked usecases

### Integration Tests

- End-to-end API tests
- Database integration tests

**Implementation Notes:**

- Create test fixtures for users and attributes
- No performance/load tests for new endpoints initially

## Implementation Order

1. Create models and constants
2. Implement repository layer
3. Implement usecase layer
4. Create API handlers and routes
5. Update FX dependency injection
6. Create default data files and initializers
7. Update chat mode model and selection logic
8. Implement Chotot integration and ListProducts tool
9. Add tests
10. Documentation and examples

## Final Implementation Notes

### Key Decisions Made:

- **Authentication**: No authentication required initially, simple ID-based access
- **User Management**: Self-registration allowed, no roles/permissions initially
- **Data Validation**: Alpha-numeric with underscores for attribute keys
- **Caching**: No external data caching initially
- **Chat Modes**: Users can customize system prompt and tools only
- **Product Service**: Standalone on-demand tool with pluggable architecture
- **Database**: Use migration files for indexes, upsert behavior for conflicts
- **Testing**: Create fixtures, no performance tests initially

The plan is ready for implementation with all questions resolved.
