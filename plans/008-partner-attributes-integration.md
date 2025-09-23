# Plan: Partner Integration User Attributes System

## Overview

Implement a system to collect and manage partner integration data from users. This includes Chotot partner data (chotot_id, chotot_oid) and WhatsApp partner data (whatsapp_phone_number_id, whatsapp_system_token). The system will show a popup after login to collect this data, display existing data as read-only with edit capabilities, and provide generic APIs to update user attributes securely.

**Key Requirements from Feedback:**

- Partner attributes are optional - users can select which partners to configure
- Popup shows on first login for new users, then accessible via user avatar dropdown
- Form modal for editing with "Skip for now" option
- Use existing UserAttribute system with tags and "sensitive" marker for encryption
- AES-256 encryption for whatsapp_system_token stored in database
- Users can only set their own attributes

## Backend Implementation

### 1. Create Encryption Utility

- Add `pkg/crypto/` package with AES-256 encryption functions
- Implement `Encrypt()` and `Decrypt()` functions using AES-256-GCM
- Read encryption key from environment variable
- Handle encryption errors gracefully

### 2. Create Partner Attributes Models & DTOs

- Add `PartnerAttributesRequest` and `PartnerAttributesResponse` models in `internal/models/user.go`
- Define constants for partner attribute keys and tags:
  - Chotot: `chotot_id`, `chotot_oid` with tag `["chotot"]`
  - WhatsApp: `whatsapp_phone_number_id`, `whatsapp_system_token` with tags `["whatsapp", "sensitive"]`

### 3. Extend User UseCase

- Add `GetPartnerAttributes(userID)` method to fetch user's partner attributes
- Add `UpdatePartnerAttributes(userID, attrs)` method to update user's partner integration data
- Filter out attributes with "sensitive" tag in GetPartnerAttributes response
- Implement encryption for attributes tagged as "sensitive"
- Use existing UserAttribute repository with Upsert for updates

### 4. Add Profile Controller Endpoints

- `GET /api/v1/profile/attributes` - Return user's partner attributes (excluding sensitive tokens)
- `PUT /api/v1/profile/attributes` - Update user's partner integration data
- Implement proper validation and error handling

### 5. Update Login Response

- Modify `LoginResponse` to include a flag indicating if partner attributes are configured
- Add `has_partner_attributes` boolean field to help frontend decide when to show popup
- Check if user has any partner attributes (chotot or whatsapp tags) to set this flag

## Frontend Implementation

### 1. Create Header Component

- Add `Header.tsx` component with user avatar dropdown
- Avatar dropdown includes "Partner Settings" option
- Integrate header into Layout component

### 2. Create Partner Attributes Components

- `PartnerAttributesModal.tsx` - Modal popup for collecting/editing partner data
- `PartnerAttributesForm.tsx` - Form component with partner selection and validation
- Partner selection: Users can choose which partners to configure (Chotot, WhatsApp)
- Handle read-only display with edit toggle functionality
- Include "Skip for now" button in modal

### 3. Extend API Client

- Add `getPartnerAttributes()` and `updatePartnerAttributes()` methods to `api.ts`
- Include proper TypeScript types for request/response

### 4. Update Login Flow

- Modify `LoginPage.tsx` to check if partner attributes popup should be shown
- Show popup after successful login if user hasn't configured partner attributes yet
- Remove refresh token attribute checking (as per feedback)

### 5. Add UI Features

- Partner selection interface with checkboxes or tabs
- Form sections for selected partners:
  - Chotot: chotot_id, chotot_oid
  - WhatsApp: whatsapp_phone_number_id, whatsapp_system_token (password input)
- Edit mode toggle for existing data
- Proper validation and error handling
- Loading states and success feedback

## Files to Create/Modify

### Backend:

- `pkg/crypto/crypto.go` (new - AES-256 encryption utility)
- `internal/models/user.go` (extend with partner attribute models and constants)
- `internal/usecase/user_usecase.go` (add partner attribute methods)
- `internal/server/controller.go` (add profile endpoints)
- `internal/server/server.go` (register new routes)

### Frontend:

- `web/src/components/Header.tsx` (new - header with user avatar dropdown)
- `web/src/components/PartnerAttributesModal.tsx` (new)
- `web/src/components/PartnerAttributesForm.tsx` (new)
- `web/src/components/Layout.tsx` (integrate header)
- `web/src/types/index.ts` (extend with partner attribute types)
- `web/src/utils/api.ts` (add partner attribute endpoints)
- `web/src/pages/LoginPage.tsx` (integrate popup)

### Documentation:

- `plans/008-partner-attributes-integration.md` (this plan)

## Security Considerations

- Never expose `whatsapp_system_token` in API responses (filter out "sensitive" tagged attributes)
- Implement AES-256-GCM encryption for attributes tagged as "sensitive"
- Store encryption key in environment variable
- Implement proper validation for all partner attribute fields
- Ensure authenticated access only for all partner attribute endpoints
- Users can only modify their own attributes

## Success Criteria

- Users can select which partners to configure (optional configuration)
- Popup shows after first login with "Skip for now" option
- User avatar dropdown provides access to partner configuration later
- Existing data displays as read-only with edit capability via modal
- System token remains encrypted and hidden from client responses
- Clean integration with existing UserAttribute system using tags
- Proper validation and error handling throughout

## Implementation Notes

**Resolved Requirements:**

- **Popup Trigger**: Show on first login for new users, accessible later via user avatar dropdown
- **Partner Selection**: Users choose which partners to configure (both optional)
- **Storage Strategy**: Use UserAttribute with tags: `["chotot"]`, `["whatsapp"]`, `["whatsapp", "sensitive"]`
- **Encryption**: AES-256 for attributes with "sensitive" tag, key from environment
- **UI Flow**: Form modal for editing with "Skip for now" option
- **Access Control**: Users can only set their own attributes
- **Error Handling**: Show error messages in modal, no automatic retries

**Technical Decisions:**

- Remove refresh token integration (as per feedback)
- Use existing UserAttribute repository with Upsert operations
- Filter sensitive attributes in API responses
- No initial validation rules for attribute formats
- No draft data persistence in forms
