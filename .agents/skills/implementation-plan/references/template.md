# Implementation Plan Template

Copy and customize this template for new implementation plans.

---

```markdown
<!--
 Copyright (c) {YEAR} {Author}. All rights reserved.
 SPDX-License-Identifier: Proprietary
-->

# {Feature Name} Implementation Plan

Implementation checklist for `specs/{feature}.md`. Each item cites the relevant specification section and source code to modify.

---

## Phase 1: Foundation

**Reference:** [{feature}.md §2](./{feature}.md#2-architecture)

**Status:** 🔲 Not Started

**Goal:** Set up project structure, dependencies, and core types.

### Assumptions

- {List key assumptions that affect implementation}
- {E.g., "Single-user mode for MVP"}

### Tasks

- [ ] Create `path/to/module/Cargo.toml` or `package.json`
  - Dependencies: {list key deps}
  - See [{feature}.md §15](./{feature}.md#15-dependencies)

- [ ] Create `path/to/module/src/lib.rs` or `index.ts`
  - Re-export all public types

- [ ] Create `path/to/module/src/types.ts`
  - `TypeA` struct/interface with fields
  - `TypeB` enum with variants
  - See [{feature}.md §3](./{feature}.md#3-data-model)

- [ ] Create `path/to/module/src/error.ts`
  - Error types using appropriate error handling pattern
  - Pattern: follow `existing/error.ts`

- [ ] Add to workspace/project config

**Verification:**
- [ ] `npm run build` or `cargo build` succeeds
- [ ] Types are importable from the module

---

## Phase 2: Data Layer

**Reference:** [{feature}.md §6](./{feature}.md#6-database-schema)

**Status:** 🔲 Not Started

**Goal:** Set up database schema, migrations, and repository layer.

- [ ] Create migration `migrations/NNN_{feature}.sql`
  - `{table_name}` table with columns
  - All indexes as specified
  - Pattern: follow `migrations/001_initial.sql`

- [ ] Create `repository/queries/{feature}.sql`
  - `Create{Entity}`, `Get{Entity}ByID`, `List{Entities}`
  - `Update{Entity}`, `Delete{Entity}`

- [ ] Generate database code (if using codegen)
  - Run `sqlc generate` or equivalent

- [ ] Create `repository/{feature}_repository.ts`
  - Repository interface with all CRUD operations
  - Implementation wrapping generated code

**Verification:**
- [ ] Migrations apply to fresh database
- [ ] Repository methods compile without errors

---

## Phase 3: Business Logic

**Reference:** [{feature}.md §7](./{feature}.md#7-business-logic)

**Status:** 🔲 Not Started

**Goal:** Implement core business logic and services.

- [ ] Create `service/{feature}_service.ts`
  - `{Feature}Service` class/struct
  - Core methods per spec
  - Input validation
  - Error handling

- [ ] Implement state machine (if applicable)
  - Valid transitions per [{feature}.md §7.1](./{feature}.md#71-state-machine)
  - Transition validation functions

- [ ] Add unit tests
  - `service/{feature}_service.test.ts`
  - Test happy paths and error cases

**Verification:**
- [ ] Unit tests pass
- [ ] State transitions work correctly

---

## Phase 4: API Layer

**Reference:** [{feature}.md §5](./{feature}.md#5-api-design)

**Status:** 🔲 Not Started

**Goal:** Implement HTTP handlers and wire up routes.

- [ ] Create `handlers/{feature}.ts`
  - `POST /api/{feature}` - create
  - `GET /api/{feature}` - list
  - `GET /api/{feature}/{id}` - get by ID
  - `PUT /api/{feature}/{id}` - update
  - `DELETE /api/{feature}/{id}` - delete

- [ ] Add request/response types
  - Request validation
  - Response serialization

- [ ] Mount routes in server
  - Add middleware (auth, logging)
  - Register handlers

- [ ] Add handler tests
  - Request validation tests
  - Response format tests

**Verification:**
- [ ] Endpoints respond with correct status codes
- [ ] Invalid requests return 400
- [ ] Authentication is enforced

---

## Phase 5: Integration

**Reference:** [{feature}.md §2](./{feature}.md#2-architecture)

**Status:** 🔲 Not Started

**Goal:** Wire all components together and verify end-to-end.

- [ ] Update main server initialization
  - Initialize service with dependencies
  - Pass to handlers

- [ ] Add integration tests
  - Full flow from API to database
  - Error scenarios

- [ ] Update configuration
  - Add new environment variables
  - Update config documentation

**Verification:**
- [ ] Integration tests pass
- [ ] Can exercise full flow via API

---

## Phase 6: Polish

**Reference:** [{feature}.md §10](./{feature}.md#10-error-handling)

**Status:** 🔲 Not Started

**Goal:** Handle edge cases, improve error messages, add logging.

- [ ] Add comprehensive error handling
  - User-friendly error messages
  - Appropriate error codes

- [ ] Add logging
  - Key operations logged
  - No sensitive data in logs

- [ ] Add documentation
  - Inline code comments for complex logic
  - Update API documentation

**Verification:**
- [ ] Error messages are helpful
- [ ] Logs show operation flow

---

## Phase 7: Testing

**Reference:** [{feature}.md §12](./{feature}.md#12-testing)

**Status:** 🔲 Not Started

**Goal:** Ensure comprehensive test coverage.

### Unit Tests

- [ ] `types.test.ts` - type validation
- [ ] `service.test.ts` - business logic
- [ ] `repository.test.ts` - data access

### Integration Tests

- [ ] `handlers.test.ts` - API endpoints
- [ ] `flow.test.ts` - end-to-end scenarios

**Verification:**
- [ ] All tests pass with race detector (if applicable)
- [ ] Coverage meets minimum threshold

---

## Files to Create

```
path/to/feature/
├── src/
│   ├── index.ts
│   ├── types.ts
│   ├── error.ts
│   ├── service.ts
│   └── repository.ts
├── handlers/
│   └── feature.ts
├── migrations/
│   └── NNN_feature.sql
└── tests/
    ├── service.test.ts
    └── handlers.test.ts
```

---

## Files to Modify

| File | Change |
|------|--------|
| `server.ts` | Add feature routes, initialize service |
| `config.ts` | Add feature configuration |
| `package.json` | Add dependencies |
| `tsconfig.json` | Add path mappings if needed |

---

## Verification Checklist

After implementation:

- [ ] `npm run build` succeeds
- [ ] `npm test` passes
- [ ] `npm run lint` clean
- [ ] Migrations run on fresh database
- [ ] Feature works end-to-end via API
- [ ] Error cases handled gracefully

---

## Decisions Made

| Question | Decision |
|----------|----------|
| {Open question from planning} | {How it was resolved} |
| {Trade-off considered} | {Which option chosen and why} |
```
