# Spec Patterns Library

Common patterns extracted from 40+ production specifications. Use these as building blocks when writing specs.

---

## Table of Contents

1. [SDK Design Patterns](#sdk-design-patterns)
2. [API Key Patterns](#api-key-patterns)
3. [Audit Event Patterns](#audit-event-patterns)
4. [Permission Table Patterns](#permission-table-patterns)
5. [SSE Streaming Patterns](#sse-streaming-patterns)
6. [Identity Resolution Patterns](#identity-resolution-patterns)
7. [Kill Switch Patterns](#kill-switch-patterns)
8. [Scoping & Precedence Patterns](#scoping--precedence-patterns)
9. [Evaluation Engine Patterns](#evaluation-engine-patterns)
10. [Stale Detection Patterns](#stale-detection-patterns)

---

## SDK Design Patterns

### Client Builder Pattern

```rust
let client = {Client}::builder()
    .api_key("{prefix}_xxx")
    .base_url("https://example.com")
    .flush_interval(Duration::from_secs(10))
    .build()?;
```

### SDK Behavior Table

| Aspect | Behavior |
|--------|----------|
| **Initialization** | Fetch initial state, start SSE connection |
| **Caching** | All data cached locally |
| **Updates** | SSE pushes updates, cache refreshed |
| **Offline** | Use last cached values |
| **Defaults** | Config default, caller can override |
| **Reconnection** | Exponential backoff on disconnect |

### HTTP Client Requirements

Both SDKs use shared HTTP client libraries:
- Rust: `loom-http` with retry, User-Agent
- TypeScript: `@loom/http` with retry, User-Agent

---

## API Key Patterns

### Key Type Table

| Type | Prefix | Use Case | Capabilities |
|------|--------|----------|--------------|
| Write | `loom_{sys}_write_` | Client-side, public | Write-only operations |
| ReadWrite | `loom_{sys}_rw_` | Server-side, secret | All operations |
| Server | `loom_{sys}_server_` | Backend services | Evaluate for any context |
| Client | `loom_{sys}_client_` | Browser, mobile | Single user context |

### Key Format

```
loom_{system}_{type}_{env}_{random}

Examples:
loom_analytics_write_7a3b9f2e1c4d8a5b6e0f3c2d1a4b5c6d7e8f9a0b
loom_flags_server_prod_8b4c0g3f2d5e9a6c7f1g4d3e2b5a6c7d8e9f0a1b
```

### Key Storage Type

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiKey {
    pub id: ApiKeyId,
    pub org_id: OrgId,
    pub environment_id: Option<EnvironmentId>,  // If environment-scoped
    pub key_type: KeyType,
    pub name: String,
    pub key_hash: String,             // Argon2 hash
    pub created_by: UserId,
    pub created_at: DateTime<Utc>,
    pub last_used_at: Option<DateTime<Utc>>,
    pub revoked_at: Option<DateTime<Utc>>,
}
```

### Key Management Rules

- Keys stored as Argon2 hashes
- Shown once at creation
- One or more keys per environment/scope allowed
- Last used timestamp tracked

---

## Audit Event Patterns

### Event Naming Convention

```
{Entity}{Action}

Examples:
FlagCreated
FlagUpdated
FlagArchived
FlagRestored
KillSwitchActivated
SdkKeyRevoked
```

### Audit Event Table

| Event | Description |
|-------|-------------|
| `{Entity}Created` | New {entity} created |
| `{Entity}Updated` | {Entity} metadata updated |
| `{Entity}Archived` | {Entity} archived/soft-deleted |
| `{Entity}Restored` | {Entity} restored from archive |
| `{Entity}Deleted` | {Entity} permanently deleted |
| `{Key}Created` | API key created |
| `{Key}Revoked` | API key revoked |

### Audit Integration Code Pattern

```rust
use loom_server_audit::{AuditLogger, AuditEvent};

async fn create_entity(
    audit: &AuditLogger,
    actor: &Actor,
    entity: &Entity,
) -> Result<()> {
    // ... create entity ...

    audit.log(AuditEvent::EntityCreated {
        entity_id: entity.id.clone(),
        org_id: entity.org_id.clone(),
        actor: actor.clone(),
    }).await?;

    Ok(())
}
```

---

## Permission Table Patterns

### Role-Based Access

| Action | Org Admin | Org Member | Super Admin |
|--------|-----------|------------|-------------|
| List {resources} | ✓ | ✓ (read) | ✓ (all orgs) |
| Create {resource} | ✓ | ✗ | ✓ (platform) |
| Update {resource} | ✓ | ✗ | ✓ (platform) |
| Archive {resource} | ✓ | ✗ | ✓ (platform) |
| Delete {resource} | ✗ | ✗ | ✓ |

### API Key-Based Access

| Action | Write Key | ReadWrite Key |
|--------|-----------|---------------|
| Capture/write | ✓ | ✓ |
| Query/read | ✗ | ✓ |
| Export | ✗ | ✓ |

### Special Permission Pattern

For high-risk operations, define separate permissions:

| Action | Permission Required |
|--------|---------------------|
| Activate kill switch | `killswitch:activate` |
| Force delete | `admin:force_delete` |
| Export PII | `data:export_pii` |

---

## SSE Streaming Patterns

### Connection Endpoint

```
GET /api/{system}/stream
Authorization: Bearer {sdk_key}
```

### Event Types Table

| Event | Description |
|-------|-------------|
| `init` | Full state of all data on connect |
| `{entity}.updated` | Entity changed |
| `{entity}.created` | New entity created |
| `{entity}.deleted` | Entity deleted |
| `heartbeat` | Keep-alive (every 30s) |

### Event Format

```json
{
  "event": "{entity}.updated",
  "data": {
    "{entity}_key": "unique.key",
    "environment": "prod",
    "enabled": true,
    "timestamp": "2026-01-10T12:00:00Z"
  }
}
```

### Reconnection Behavior

- SDK should reconnect with exponential backoff
- On reconnect, server sends `init` event with full state
- Client clears local cache and replaces with init data

---

## Identity Resolution Patterns

### Identity Model (PostHog-Style)

1. **Anonymous user arrives**: SDK generates UUIDv7, stored in localStorage/cookie
2. **Events captured**: All events tagged with this `distinct_id`
3. **User identifies**: SDK calls `identify(anonymous_id, user_id)`
4. **Merge occurs**: Both distinct_ids linked to same Person
5. **Future events**: Can use either distinct_id

### Identify Payload

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IdentifyPayload {
    pub distinct_id: String,          // Current (often anonymous)
    pub user_id: String,              // The "real" identifier
    #[serde(default)]
    pub properties: serde_json::Value, // Properties to set
}
```

### Merge Rules

1. Person with `Identified` type wins over `Anonymous`
2. If both identified, older Person wins
3. All events from loser reassigned to winner
4. All identities from loser moved to winner
5. Properties merged (winner takes precedence)
6. Loser marked as merged (soft delete)

### PersonIdentity Type

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PersonIdentity {
    pub id: PersonIdentityId,
    pub person_id: PersonId,
    pub distinct_id: String,
    pub identity_type: IdentityType,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum IdentityType {
    Anonymous,    // SDK-generated UUIDv7
    Identified,   // User's real ID
}
```

---

## Kill Switch Patterns

### Kill Switch Design

| Aspect | Behavior |
|--------|----------|
| **Targeting** | Global on/off only — no strategies |
| **Linked resources** | When activated, forces all linked to off/default |
| **Activation reason** | Required field when activating |
| **Reset** | Manual only — no auto-reset |
| **Permissions** | Separate permission required |
| **Priority** | Evaluated before normal logic |

### Kill Switch Type

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KillSwitch {
    pub id: KillSwitchId,
    pub org_id: Option<OrgId>,        // None = platform-level
    pub key: String,
    pub name: String,
    pub description: Option<String>,
    pub linked_keys: Vec<String>,     // Keys of linked resources
    pub is_active: bool,
    pub activated_at: Option<DateTime<Utc>>,
    pub activated_by: Option<UserId>,
    pub activation_reason: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}
```

### Activation Flow

```
1. User with activation permission activates kill switch
2. Required: activation_reason field
3. Kill switch marked active with timestamp and user
4. All linked resources immediately affected
5. SSE broadcast to all connected clients
6. Audit log entry created
```

---

## Scoping & Precedence Patterns

### Two-Tier Scoping

| Scope | Managed By | Use Case |
|-------|------------|----------|
| **Platform** | Super admins | Global features, maintenance |
| **Organization** | Org admins | Org-specific features |

### Precedence Rules

1. **Platform overrides org** — Platform config takes precedence
2. **Kill switches override normal** — Active kill switch forces default
3. **Platform kill switches override all** — Affects all orgs
4. **Prerequisites first** — Missing prerequisite returns default

### Key Format Validation

Structured dot-notation: `{domain}.{feature}[.{sub}]`

Examples:
- `checkout.new_flow`
- `billing.subscription.annual`

Validation:
- Lowercase alphanumeric with dots and underscores
- 3-100 characters
- Cannot start or end with dot
- Pattern: `^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`

---

## Evaluation Engine Patterns

### Evaluation Order

```
1. Check if resource exists
   → Not found: return error or SDK-provided default

2. Check environment config (if applicable)
   → Disabled: return default with reason=Disabled

3. Check kill switches (platform first, then org)
   → Active: return default with reason=KillSwitch

4. Check prerequisites (if applicable)
   → Missing: return default with reason=Prerequisite

5. Check strategy/rules (if configured)
   → Evaluate conditions
   → Apply percentage (if configured)
   → Return result based on strategy

6. No strategy or conditions not met
   → Return default with reason=Default
```

### Evaluation Result Type

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationResult {
    pub key: String,
    pub value: Value,
    pub reason: EvaluationReason,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvaluationReason {
    Default,
    Strategy { strategy_id: StrategyId },
    KillSwitch { kill_switch_id: KillSwitchId },
    Prerequisite { missing: String },
    Disabled,
    Error { message: String },
}
```

### Percentage Hashing

Consistent hashing for sticky assignment:

```rust
fn evaluate_percentage(key: &str, resource_key: &str, percentage: u32) -> bool {
    let input = format!("{}.{}", resource_key, key);
    let hash = murmur3_32(&input, 0);
    let bucket = hash % 100;
    bucket < percentage
}
```

---

## Stale Detection Patterns

### Staleness Criteria

A resource is considered stale if:
- Not accessed in the last N days (e.g., 30)
- No configuration changes in the last M days (e.g., 90)

### Stats Tracking Type

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceStats {
    pub resource_id: ResourceId,
    pub last_accessed_at: Option<DateTime<Utc>>,
    pub access_count_24h: u64,
    pub access_count_7d: u64,
    pub access_count_30d: u64,
}
```

### Stale Detection Query

```sql
SELECT
    r.id,
    r.key,
    rs.last_accessed_at,
    rs.access_count_30d
FROM resources r
LEFT JOIN resource_stats rs ON rs.resource_id = r.id
WHERE r.archived_at IS NULL
  AND (
    rs.last_accessed_at IS NULL
    OR rs.last_accessed_at < datetime('now', '-30 days')
  )
ORDER BY rs.last_accessed_at ASC NULLS FIRST;
```

---

## Tracking Helper Patterns (Web)

### Helper Function Signatures

| Function | Parameters | Event Name |
|----------|------------|------------|
| `trackLinkClick` | `linkName, href, properties?` | `link_clicked` |
| `trackButtonClick` | `buttonName, properties?` | `button_clicked` |
| `trackFormSubmit` | `formName, properties?` | `form_submitted` |
| `trackModalOpen` | `modalName, properties?` | `modal_opened` |
| `trackModalClose` | `modalName, properties?` | `modal_closed` |
| `trackFilterChange` | `filterName, value, properties?` | `filter_changed` |
| `trackAction` | `action, resourceType, resourceId, properties?` | `action_performed` |

### Property Conventions

| Property | Description | Example |
|----------|-------------|---------|
| `link_name` | Descriptive link identifier | `'project_card'` |
| `button_name` | Descriptive button identifier | `'create_weaver'` |
| `form_name` | Form identifier | `'create_org'` |
| `modal_name` | Modal identifier | `'delete_confirmation'` |
| `filter_name` | Filter identifier | `'status'` |
| `action` | Action performed | `'resolve'` |
| `resource_type` | Type of resource | `'issue'` |
| `resource_id` | Resource identifier | `'issue-abc'` |

---

## Environment Patterns

### Auto-Created Environments

When an organization is created, auto-provision:

| Environment | Description |
|-------------|-------------|
| `dev` | Development environment |
| `prod` | Production environment |

Org admins can create additional (e.g., `staging`, `qa`).

### Environment-Scoped Configuration

- Each resource has separate config per environment
- When resource created, config auto-created for all environments
- SDK keys scoped to single environment
- SSE streams are environment-specific

---

## Secret Handling Patterns

### Using loom-secret

```rust
use loom_secret::Secret;

pub struct Event {
    // ...
    pub ip_address: Option<Secret<String>>,  // Auto-redacts in logs
}

// To access the value:
let ip = event.ip_address.as_ref().map(|s| s.expose());
```

### What to Wrap

- API keys
- Tokens
- Passwords
- IP addresses
- PII fields

### Benefits

- Auto-redaction in Debug output
- Auto-redaction in tracing
- Explicit `.expose()` required to access
- Serialization can be controlled
