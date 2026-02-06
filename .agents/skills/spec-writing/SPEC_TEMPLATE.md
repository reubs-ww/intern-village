# Spec Template

Copy-paste template for creating new specifications. Replace all `{placeholders}` with actual values. Remove sections that don't apply.

**Remember:** This spec must be **standalone**. A fresh agent should be able to implement this feature without reading other documents or having prior context.

---

```markdown
<!--
 Copyright (c) {YEAR} {Owner}. All rights reserved.
 SPDX-License-Identifier: {License}
-->

# {Feature Name} Specification

**Status:** Planned | In Progress | Implemented
**Version:** 1.0
**Last Updated:** {YYYY-MM-DD}
**Architecture Reference:** [architecture.md](./architecture.md)

---

## 1. Overview

### Purpose

{One paragraph explaining WHAT this feature does and WHY it exists. Include enough context that a reader unfamiliar with the project understands the problem being solved.}

### Goals

- **{Goal 1}**: {Specific, measurable outcome}
- **{Goal 2}**: {Specific, measurable outcome}
- **{Goal 3}**: {Specific, measurable outcome}

### Non-Goals

Explicit boundaries — what this feature does NOT do:

- {Feature X} — out of scope, handled by {other feature}
- {Capability Y} — deferred to future version
- {Integration Z} — handled by {other system}

### Context from Architecture

Briefly summarize relevant architectural decisions that affect this spec:

- {Decision 1}: {How it affects this feature}
- {Decision 2}: {How it affects this feature}

---

## 2. Package/Module Structure

Where the code for this feature lives.

```
{project-root}/
├── {feature}/
│   ├── types.{ext}              # Core types/models
│   ├── repository.{ext}         # Data access layer
│   ├── service.{ext}            # Business logic
│   ├── handlers.{ext}           # API handlers
│   └── errors.{ext}             # Error types
├── sdk/                         # Client SDK (if applicable)
│   └── {feature}/
│       ├── client.{ext}
│       └── types.{ext}
```

---

## 3. User Flows

How users interact with this feature.

### Flow 1: {Name}

1. {User action}
2. {System response}
3. {Result user sees}

### Flow 2: {Name}

1. {User action}
2. {System response}
3. {Result user sees}

---

## 4. Data Model

### 4.1 {Entity Name}

{Description of what this entity represents and when it's used.}

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (UUID) | Yes | Unique identifier |
| {field_name} | {type} | {Yes/No} | {What this field stores} |
| {field_name} | {type} | {Yes/No} | {What this field stores} |
| created_at | datetime | Yes | Creation timestamp |
| updated_at | datetime | Yes | Last modification timestamp |

**Relationships:**
- Belongs to: {Parent entity} (via {foreign_key})
- Has many: {Child entity}

**Constraints:**
- {Unique constraint, validation rule, etc.}

### 4.2 {Another Entity}

{Repeat pattern above}

---

## 5. API Design

### Base Path

All endpoints under `/api/{feature}/*`

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/{feature}` | Required | List all {resources} |
| POST | `/api/{feature}` | Required | Create {resource} |
| GET | `/api/{feature}/{id}` | Required | Get {resource} by ID |
| PATCH | `/api/{feature}/{id}` | Required | Update {resource} |
| DELETE | `/api/{feature}/{id}` | Required | Delete {resource} |

### Request/Response Examples

#### Create {Resource}

**Request:**
```json
POST /api/{feature}
Content-Type: application/json

{
  "{field}": "{value}",
  "{field}": "{value}"
}
```

**Response (201 Created):**
```json
{
  "id": "{uuid}",
  "{field}": "{value}",
  "{field}": "{value}",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

#### Get {Resource}

**Request:**
```
GET /api/{feature}/{id}
```

**Response (200 OK):**
```json
{
  "id": "{uuid}",
  "{field}": "{value}",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

#### List {Resources}

**Request:**
```
GET /api/{feature}?{query_param}={value}
```

**Response (200 OK):**
```json
{
  "items": [...],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | INVALID_REQUEST | Request body validation failed |
| 401 | UNAUTHORIZED | Missing or invalid authentication |
| 403 | FORBIDDEN | User lacks permission |
| 404 | NOT_FOUND | Resource not found |
| 409 | CONFLICT | Resource already exists |
| 500 | INTERNAL_ERROR | Unexpected server error |

---

## 6. Database Schema

### Table: {table_name}

```sql
CREATE TABLE {table_name} (
    id TEXT PRIMARY KEY,
    {column} {TYPE} {NOT NULL} {DEFAULT},
    {column} {TYPE} {NOT NULL} {DEFAULT},
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign keys
    FOREIGN KEY ({column}) REFERENCES {other_table}(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_{table_name}_{column} ON {table_name}({column});
CREATE UNIQUE INDEX idx_{table_name}_{unique_column} ON {table_name}({column});
```

### Table: {another_table}

{Repeat pattern above}

### Migration Notes

- Migration file: `migrations/{NNN}_{feature_name}.sql`
- {Any special considerations for migration}

---

## 7. Business Logic

### 7.1 {Operation Name}

{What this operation does and when it's triggered.}

**Trigger:** {When this happens}

**Inputs:**
- {Input 1}: {Description}
- {Input 2}: {Description}

**Steps:**
1. Validate {what}
2. Check {precondition}
3. Perform {action}
4. Update {state}
5. Return {result}

**Output:** {What is returned}

**Edge Cases:**

| Scenario | Behavior |
|----------|----------|
| {Edge case 1} | {What happens} |
| {Edge case 2} | {What happens} |
| {Error condition} | {Error returned} |

### 7.2 {Another Operation}

{Repeat pattern above}

---

## 8. External Integrations

### {Service/Tool Name}

**Purpose:** {Why we integrate with this}

**Operations Used:**

| Operation | When | How |
|-----------|------|-----|
| {Op 1} | {Trigger} | {Method/endpoint} |
| {Op 2} | {Trigger} | {Method/endpoint} |

**Authentication:** {How we auth with this service}

**Error Handling:**
- {Error scenario}: {How we handle it}
- Service unavailable: {Fallback behavior}

---

## 9. SDK Design (if applicable)

If this feature has a client SDK, document it here.

### SDK Usage Example

```{language}
// Initialize
const client = new {Feature}Client({
  apiKey: '{prefix}_xxx',
  baseUrl: 'https://api.example.com',
});

// Main operation
await client.{operation}('{param}', {
  key: 'value',
});

// Cleanup (if needed)
await client.shutdown();
```

### SDK Behavior

| Aspect | Behavior |
|--------|----------|
| **Initialization** | {What happens on init} |
| **Batching** | {How requests are batched, if at all} |
| **Retry** | {Retry strategy on failure} |
| **Offline** | {How offline scenarios are handled} |
| **Persistence** | {What is persisted locally, if anything} |

### API Key Format

| Type | Prefix | Use Case | Capabilities |
|------|--------|----------|--------------|
| {Type 1} | `{prefix}_` | {Where used} | {What it can do} |
| {Type 2} | `{prefix}_` | {Where used} | {What it can do} |

---

## 10. Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `{VAR_NAME}` | {type} | {Yes/No} | `{default}` | {What it controls} |
| `{VAR_NAME}` | {type} | {Yes/No} | `{default}` | {What it controls} |

---

## 11. Audit Events (if applicable)

What actions are logged for audit/compliance purposes.

| Event | When Logged | Data Captured |
|-------|-------------|---------------|
| `{Feature}{Action}` | {Trigger} | {What data is recorded} |
| `{Feature}{Action}` | {Trigger} | {What data is recorded} |

---

## 12. Security Considerations

### Authentication

{How users authenticate to use this feature}

### Authorization

| Action | Who Can Perform |
|--------|-----------------|
| Create | {Roles/conditions} |
| Read | {Roles/conditions} |
| Update | {Roles/conditions} |
| Delete | {Roles/conditions} |

### Data Protection

- **Sensitive data**: {What sensitive data exists}
- **Protection**: {How it's protected - encryption, hashing, etc.}
- **Audit**: {What actions are logged}

### Threats Considered

| Threat | Mitigation |
|--------|------------|
| {Threat 1} | {How we prevent it} |
| {Threat 2} | {How we prevent it} |

---

## 13. Testing Requirements

### Unit Tests

- [ ] {Component} validates {input}
- [ ] {Component} handles {edge case}
- [ ] {Component} returns {expected output}

### Integration Tests

- [ ] API endpoint {X} returns correct response for {scenario}
- [ ] Database operations persist correctly
- [ ] {Integration point} works end-to-end

### Manual Verification

- [ ] {User flow 1} works as expected
- [ ] {User flow 2} works as expected
- [ ] Error messages are clear and helpful

---

## 14. Implementation Phases

### Phase 1: Data Layer

Files to create/modify:
- `{path/to/migration.sql}` - Database migration
- `{path/to/repository}` - Data access layer

Tasks:
- [ ] Create database migration
- [ ] Implement repository with CRUD operations
- [ ] Add data validation

### Phase 2: Business Logic

Files to create/modify:
- `{path/to/service}` - Business logic

Tasks:
- [ ] Implement {operation 1}
- [ ] Implement {operation 2}
- [ ] Add error handling

### Phase 3: API Layer

Files to create/modify:
- `{path/to/handlers}` - HTTP handlers
- `{path/to/routes}` - Route definitions

Tasks:
- [ ] Create API endpoints
- [ ] Add request validation
- [ ] Add response formatting

### Phase 4: Integration

Files to modify:
- `{path/to/main}` - Wire up new routes

Tasks:
- [ ] Register routes in application
- [ ] Add middleware if needed
- [ ] Verify end-to-end flow

### Phase 5: Testing

Files to create:
- `{path/to/tests}` - Test files

Tasks:
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Manual verification

### Verification Checklist

After implementation, verify:
- [ ] All tests pass
- [ ] Linting passes
- [ ] All user flows work
- [ ] Error handling works correctly

---

## 15. Dependencies

### Internal Dependencies

| Module | What We Use |
|--------|-------------|
| {existing module} | {Functions/types we depend on} |

### External Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| {package} | {version} | {Why needed} |

---

## Appendix A: {Topic}

{Detailed information, examples, or context that doesn't fit in main sections but is useful for implementation.}

---

## Appendix B: Compatibility Notes (if applicable)

If this feature follows patterns from external systems (e.g., PostHog, Stripe), document differences:

| External System | This System | Notes |
|-----------------|-------------|-------|
| {External feature} | {Our equivalent} | {Differences or same semantics} |

---

## Appendix C: Future Considerations

| Feature | Why Deferred | Potential Approach |
|---------|--------------|-------------------|
| {Future 1} | {Reason} | {How we might implement later} |
| {Future 2} | {Reason} | {How we might implement later} |
```

---

## Usage Notes

1. **Replace all `{placeholders}`** with actual values
2. **Remove sections** that don't apply to your feature
3. **Add sections** if your feature has unique concerns (e.g., "Identity Resolution" in the sample)
4. **Keep examples concrete** — use realistic data, not lorem ipsum
5. **Be exhaustive on data model and API** — agents can't invent what's missing
6. **Make phases small and testable** — each should be independently verifiable
7. **Update `specs/README.md`** after creating the spec
