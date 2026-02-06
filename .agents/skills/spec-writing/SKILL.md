---
name: spec-writing
description: Guide for writing detailed implementation specifications with types, APIs, schemas, and implementation phases. Use AFTER architecture is defined. For high-level system design (components, decisions, trade-offs), use the architect skill instead.
---

# Spec Writing Skill

Comprehensive guide for writing **detailed implementation specifications** that agents can use to implement features autonomously.

## Purpose

Create specifications that serve as **authoritative implementation blueprints**. These are detailed documents with types, APIs, schemas, and phased implementation steps.

**CRITICAL**: Specs must be **self-contained**. A fresh agent with no prior context should be able to read a spec and implement the feature without needing the original conversation or additional clarification.

## When to Use This Skill

Use this skill when:
- User asks to "write a spec for X" (a specific feature, not overall architecture)
- User says "spec out a feature"
- You need to document detailed types, APIs, or database schemas
- **Architecture already exists** and you need implementation details

## When NOT to Use This Skill

**Use the `architect` skill instead when:**
- User asks for "system architecture" or "high-level design"
- User wants to understand component relationships and trade-offs
- User says "design the system for..." or "how should I structure..."
- Focus is on DECISIONS and TRADE-OFFS, not implementation details

## Key Differences

| Aspect | Architect | Spec |
|--------|-----------|------|
| **Focus** | WHAT and WHY (decisions) | WHAT in detail (blueprint) |
| **Abstraction** | High-level | Detailed |
| **Code** | None | Types, schemas, API examples |
| **Output** | `specs/architecture.md` | `specs/{feature}.md` |
| **Quantity** | 1 per project | 1 per feature/component |

---

## Core Principles

### Specs Must Stand Alone

A spec is complete when a fresh agent can:
1. Understand the PURPOSE without reading other docs
2. Know the exact TYPES and SCHEMAS to create
3. See the API CONTRACTS to implement
4. Follow the PHASES step-by-step
5. Verify success with clear CRITERIA

**Include everything. Assume the reader has no context.**

### Specs Enable Autonomous Implementation

Good specs let agents:
1. Understand the WHY (purpose, goals, non-goals)
2. Know the WHAT (types, APIs, schemas)
3. Follow the HOW (implementation phases)
4. Avoid mistakes (edge cases, security considerations)

### Reference Architecture, Don't Repeat It

Link to `specs/architecture.md` for high-level decisions, but include enough context that the reader doesn't need to read it to understand this spec.

---

## Workflow

### Step 1: Read Architecture First

Before writing a spec:
1. Read `specs/architecture.md`
2. Identify which component(s) this feature affects
3. Note relevant decisions and constraints

### Step 2: Clarify Scope

Ask focused questions if needed:
- What's the MVP vs nice-to-have?
- Any specific edge cases to handle?
- Integration points with existing code?

### Step 3: Write the Spec

Follow the template below. Include ALL sections that apply.

### Step 4: Review with User

Present the spec, highlight key design choices, invite feedback.

---

## Spec Document Template

```markdown
<!--
 Copyright (c) {YEAR} {Owner}. All rights reserved.
 SPDX-License-Identifier: {License}
-->

# {Feature Name} Specification

**Status:** Planned | In Progress | Implemented
**Version:** 1.0
**Last Updated:** YYYY-MM-DD
**Architecture Reference:** [architecture.md](./architecture.md)

---

## 1. Overview

### Purpose

One paragraph explaining WHAT this feature does and WHY it exists. Include enough context that a reader unfamiliar with the project understands the problem being solved.

### Goals

- **{Goal 1}**: {Specific, measurable outcome}
- **{Goal 2}**: {Specific, measurable outcome}

### Non-Goals

Explicit boundaries — what this feature does NOT do:

- {Feature X} — out of scope, handled by {other feature}
- {Capability Y} — deferred to future version

### Context from Architecture

Briefly summarize relevant architectural decisions that affect this spec:

- {Decision}: {How it affects this feature}

---

## 2. Package/Module Structure

Where the code for this feature lives. Helps agents understand the project layout.

```
src/                              # or your project's structure
├── {feature}/
│   ├── types.{ext}              # Core types/models
│   ├── repository.{ext}         # Data access layer
│   ├── service.{ext}            # Business logic
│   ├── handlers.{ext}           # API handlers
│   └── errors.{ext}             # Error types
├── sdk/
│   └── {feature}/               # Client SDK (if applicable)
```

---

## 3. User Flows

How users interact with this feature. Essential for understanding the full scope.

### Flow 1: {Name}

1. User does X
2. System responds with Y
3. User sees Z

### Flow 2: {Name}

...

---

## 3. Data Model

### 3.1 {Entity Name}

{Description of what this entity represents.}

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (UUID) | Yes | Unique identifier |
| name | string | Yes | Display name |
| status | enum | Yes | One of: pending, active, completed |
| metadata | object | No | Additional key-value data |
| created_at | datetime | Yes | Creation timestamp |
| updated_at | datetime | Yes | Last update timestamp |

**Relationships:**
- Belongs to: {Parent entity}
- Has many: {Child entities}

### 3.2 {Another Entity}

...

---

## 4. API Design

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
{
  "name": "Example",
  "status": "pending",
  "metadata": {
    "key": "value"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "uuid-here",
  "name": "Example",
  "status": "pending",
  "metadata": {
    "key": "value"
  },
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | INVALID_REQUEST | Request body validation failed |
| 404 | NOT_FOUND | Resource not found |
| 409 | CONFLICT | Resource already exists |

---

## 5. Database Schema

### Table: {table_name}

```sql
CREATE TABLE {table_name} (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_{table_name}_status ON {table_name}(status);
```

### Migrations

Migration file: `migrations/{NNN}_{feature_name}.sql`

---

## 6. Business Logic

### 6.1 {Operation Name}

**Trigger:** When {condition}

**Steps:**
1. Validate {inputs}
2. Check {preconditions}
3. Perform {action}
4. Update {state}
5. Return {result}

**Edge Cases:**

| Scenario | Behavior |
|----------|----------|
| {Edge case 1} | {What happens} |
| {Edge case 2} | {What happens} |

### 6.2 {Another Operation}

...

---

## 7. External Integrations

### {Service Name}

**Purpose:** Why we integrate with this service.

**Operations Used:**

| Operation | When | Endpoint/Method |
|-----------|------|-----------------|
| {Op 1} | {Trigger} | `POST /api/...` |
| {Op 2} | {Trigger} | `GET /api/...` |

**Authentication:** {How we authenticate}

**Error Handling:** {What happens when service is unavailable}

---

## 8. SDK Design (if applicable)

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
```

### SDK Behavior

| Aspect | Behavior |
|--------|----------|
| **Initialization** | {What happens on init} |
| **Batching** | {How requests are batched, if at all} |
| **Retry** | {Retry strategy on failure} |
| **Offline** | {How offline scenarios are handled} |
| **Persistence** | {What is persisted locally, if anything} |

### API Key Format (if applicable)

| Type | Prefix | Use Case | Capabilities |
|------|--------|----------|--------------|
| {Type 1} | `{prefix}_` | {Where used} | {What it can do} |
| {Type 2} | `{prefix}_` | {Where used} | {What it can do} |

---

## 9. Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `FEATURE_ENABLED` | boolean | No | `true` | Enable this feature |
| `FEATURE_TIMEOUT_MS` | integer | No | `5000` | Request timeout |

---

## 10. Audit Events (if applicable)

What actions are logged for audit/compliance purposes.

| Event | When Logged | Data Captured |
|-------|-------------|---------------|
| `{Feature}{Action}` | {Trigger} | {What data is recorded} |

---

## 11. Security Considerations

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

- {What sensitive data exists}
- {How it's protected}

---

## 12. Testing Requirements

### Unit Tests

- [ ] {Component} validates {behavior}
- [ ] {Component} handles {edge case}

### Integration Tests

- [ ] API endpoint returns correct response
- [ ] Database operations persist correctly

### Manual Verification

- [ ] {User flow} works end-to-end

---

## 13. Implementation Phases

### Phase 1: Data Layer

- [ ] Create database migration
- [ ] Implement repository/data access

### Phase 2: Business Logic

- [ ] Implement core operations
- [ ] Add validation logic

### Phase 3: API Layer

- [ ] Create API endpoints
- [ ] Add request/response handling

### Phase 4: Integration

- [ ] Wire up to existing system
- [ ] Add to routing

### Phase 5: Testing

- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Manual verification

---

## 14. Dependencies

### Internal

- {Existing module}: {What we use from it}

### External

| Package | Version | Purpose |
|---------|---------|---------|
| {package} | {version} | {Why needed} |

---

## Appendix A: {Topic}

{Detailed information that doesn't fit in main flow}

## Appendix B: Compatibility Notes (if applicable)

If this feature follows patterns from external systems, document differences:

| External System | This System | Notes |
|-----------------|-------------|-------|
| {External feature} | {Our equivalent} | {Differences} |

## Appendix C: Future Considerations

| Feature | Why Deferred | Potential Approach |
|---------|--------------|-------------------|
| {Future 1} | {Reason} | {How we might do it} |
```

---

## Checklist Before Finishing

Before marking a spec complete, verify:

- [ ] **Standalone**: Can a fresh agent implement this without other docs?
- [ ] **Complete user flows**: All user interactions documented?
- [ ] **Data model clear**: All entities, fields, and relationships defined?
- [ ] **API contracts**: Request/response examples for all endpoints?
- [ ] **Edge cases**: Error scenarios and handling documented?
- [ ] **Phases actionable**: Each phase has concrete, checkable tasks?

---

## Lookup Table Pattern

**Every project with specs needs a `specs/README.md` as a lookup table.**

```markdown
# Project Specifications

| Spec | Component | Status | Purpose |
|------|-----------|--------|---------|
| [architecture.md](./architecture.md) | All | Approved | System architecture |
| [auth.md](./auth.md) | Backend | Planned | Authentication system |
| [tasks.md](./tasks.md) | Backend | In Progress | Task management |
```

---

## Remember

- **Specs must stand alone** — a fresh agent should implement without needing the original conversation
- **Architecture = decisions, Specs = blueprints** — don't repeat trade-off discussions in specs
- **Be exhaustive on data and APIs** — agents can skip what's not needed, but can't invent what's missing
- **Phases should be small and verifiable** — each phase should be completable and testable independently
