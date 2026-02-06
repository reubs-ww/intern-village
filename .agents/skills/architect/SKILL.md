---
name: architect
description: Design and review system architecture with a pragmatic, right-sized approach. Produces specs/architecture.md with high-level components, data flow, and key decisions. Challenges over-engineering and clarifies actual requirements before designing. Triggers on /architect command or keywords like "design system", "architecture", "review design", "system design".
---

# Architect Skill

Design and review system architectures with a focus on **practical, right-sized solutions** for small-to-medium projects. Avoids over-engineering and trendy-but-unnecessary patterns.

## When to Use This Skill

**Explicit triggers:**
- User types `/architect`
- User types `/architect review`

**Auto-detect triggers:**
- "design the system for..."
- "architecture for..."
- "review the design"
- "how should I structure..."
- "system design for..."
- "what's the architecture for..."

## Core Philosophy

### Right-Size Everything

Before designing, always ask:
- **Does this actually need microservices?** Most projects work fine as a monolith.
- **Does this need to scale to millions?** Design for realistic load, not hypothetical traffic.
- **Is this complexity justified?** Simpler is better until proven otherwise.
- **What's the actual deployment target?** A single VPS? Kubernetes? Serverless?

### Challenge Assumptions

When reviewing or creating architecture:
- Question every abstraction layer
- Challenge every service boundary
- Ask "what problem does this solve?" for complex patterns
- Prefer boring technology that works

### Output Location

All architecture documents go to: `specs/architecture.md`

If updating existing file: preserve manual additions, update sections cleanly.

---

## Workflow

### Mode 1: New Architecture (`/architect` or "design a system for X")

**Step 1: Quick Clarification**

Before drafting, ask 2-3 focused questions:

```
Before I design this, let me understand your actual needs:

1. **Scale**: How many users/requests are you realistically expecting? (10? 1000? 100k?)
2. **Deployment**: Where will this run? (Single server, cloud VMs, containers, serverless?)
3. **Team size**: How many developers will maintain this?
```

Adjust questions based on context. Skip if answers are obvious from requirements.

**Step 2: Analyze Existing Codebase (if exists)**

If there's existing code:
- Identify current patterns and tech stack
- Note existing structure to build upon
- Don't reinvent what's already working

**Step 3: Draft Architecture**

Produce `specs/architecture.md` following the template below.

**Step 4: Review with User**

Present the draft, highlight key decisions, invite feedback.

---

### Mode 2: Review Architecture (`/architect review` or "review the design")

**Step 1: Read Existing Architecture**

Load `specs/architecture.md` or file specified by user.

**Step 2: Analyze for Issues**

Check for:
- **Over-engineering**: Is this more complex than needed?
- **Missing pieces**: Are there gaps in the design?
- **Unclear boundaries**: Are component responsibilities fuzzy?
- **Scalability mismatch**: Is it designed for wrong scale?
- **Tech choices**: Are the technology choices justified?

**Step 3: Clarify Expectations**

Ask the user about their actual requirements before suggesting changes:

```
I see the design uses [pattern X]. Before I suggest changes:
- What load are you actually expecting?
- Is [capability Y] a real requirement or a nice-to-have?
```

**Step 4: Provide Feedback**

Be specific and actionable:
- "This service boundary seems unnecessary — Component A and B could be merged"
- "The message queue adds complexity. With your expected load, direct HTTP calls would work fine"
- "Missing: how does authentication flow between services?"

---

## Architecture Document Template

**CRITICAL**: The architecture document must be **self-contained**. A fresh agent with no prior context should be able to read this document and have everything needed to spec out individual components. This means:

- Capture ALL decisions made during the discussion
- Include user flows, not just component diagrams
- Document data entities and their relationships
- Specify external integrations and their requirements
- Detail agent/worker behaviors if applicable
- List configuration requirements

**Remember**: This is an architecture doc, NOT an implementation doc. Focus on WHAT and WHY, not HOW. No code snippets, no SQL schemas, no directory structures.

```markdown
# {Project Name} Architecture

**Status:** Draft | Approved | Implemented
**Last Updated:** YYYY-MM-DD

---

## Overview

### What This System Does

One paragraph explaining the system's purpose in plain language.

### Key Constraints

- **Scale target**: X concurrent users / Y requests per second
- **Deployment**: Where and how it runs
- **Team**: Who maintains this

---

## High-Level Components

### Component Diagram

\`\`\`
(Use ASCII diagrams for portability)

┌─────────────┐     ┌─────────────┐
│  Component  │────>│  Component  │
└─────────────┘     └─────────────┘
\`\`\`

### Component Responsibilities

| Component | Responsibility | Tech |
|-----------|---------------|------|
| API Server | Handles all business logic and HTTP requests | Node.js / Go / etc |
| Database | Persistent storage | PostgreSQL / SQLite |
| Cache | Session storage, hot data | Redis / in-memory |

---

## User Flows

Document the key user journeys through the system. Each flow should be a numbered sequence of steps.

### Flow 1: {Name}

1. User does X
2. System responds with Y
3. User sees Z

### Flow 2: {Name}

...

(Include all major flows: onboarding, primary actions, error cases)

---

## Data Flow

### Primary System Flow

\`\`\`
(ASCII sequence diagram showing how data moves between components)

User          API           Database
  │            │               │
  │──Request──>│               │
  │            │───Query──────>│
  │            │<──Result──────│
  │<─Response──│               │
\`\`\`

Describe the main data flow in 2-3 sentences.

---

## Data Entities

List all entities that need to be stored, with their purpose and key fields.

| Entity | Purpose | Key Fields |
|--------|---------|------------|
| User | User accounts | id, email, name, created_at |
| Session | Active sessions | id, user_id, token, expires_at |

(Include relationships: "User has many Sessions")

---

## Key Decisions

Document every significant decision made during the design discussion.

### Decision 1: {Title}

**Context**: What situation prompted this decision?

**Decision**: What we chose and why.

**Trade-offs**: What we gave up, what we gained.

### Decision 2: {Title}

...

---

## What We're NOT Doing (and Why)

| Pattern/Feature | Why Not |
|-----------------|---------|
| Microservices | Project size doesn't justify the operational overhead |
| Kubernetes | Single server deployment is sufficient for expected load |
| Event sourcing | Simple CRUD is adequate for this use case |

---

## External Integrations

Document any external services, APIs, or tools the system integrates with.

### {Integration Name}

**Purpose**: Why we integrate with this.

**Operations**: What operations we perform (list them).

**Auth/Scopes**: What authentication or permissions are required.

(Repeat for each integration: OAuth providers, external APIs, CLI tools, etc.)

---

## Component Behaviors

For autonomous components (agents, workers, background jobs), document their behavior.

### {Component Name}

**Trigger**: What causes this component to run?

**Inputs**: What data/context does it receive?

**Responsibilities**: What does it do? (numbered list)

**Outputs**: What does it produce?

**Error Handling**: How does it handle failures?

(Repeat for each autonomous component)

---

## Configuration

What must users/operators provide to run this system?

| Config | Required | Purpose |
|--------|----------|---------|
| DATABASE_URL | Yes | Database connection |
| API_KEY | Yes | External API auth |

---

## Technology Choices

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Frontend | React | User preference, widely supported |
| Backend | Go | Good for concurrent operations |
| Database | PostgreSQL | Reliable, good enough for scale |

---

## Future Considerations

Things that might matter later but don't need solving now:

- **If traffic grows 10x**: Consider adding read replicas
- **If team grows**: Consider splitting into modules (not services)
```

---

## Anti-Patterns to Call Out

When reviewing, flag these common over-engineering patterns:

| Pattern | When It's Overkill |
|---------|-------------------|
| Microservices | < 5 developers, < 10k daily users |
| Kubernetes | Single app, no complex scaling needs |
| Event sourcing | Standard CRUD app |
| GraphQL | Simple REST would suffice |
| Message queues | Synchronous calls would work |
| Multiple databases | Single DB handles the load fine |
| CQRS | Read/write patterns aren't that different |

**Always ask**: "What problem does this complexity solve?"

---

## Integration with Codebase

When codebase exists, align architecture with:

1. **Existing tech stack** — Don't introduce new languages/frameworks without justification
2. **Current patterns** — Build on what's working
3. **Team familiarity** — Prefer technologies the team knows

---

## Updating Existing Architecture

When `specs/architecture.md` already exists:

1. Read the current document fully
2. Identify what's changing
3. Update sections in place
4. Add a changelog entry at the bottom if significant changes
5. Preserve any manual notes or TODOs the user added

---

## Remember

- **Start simple, add complexity only when proven necessary**
- **A working monolith beats a broken microservice architecture**
- **Design for today's problems, not imaginary future scale**
- **Clear documentation of a simple system > vague documentation of a complex one**
- **The architecture doc must stand alone** — a fresh agent should have ALL context needed to spec components without access to the original conversation
- **Architecture = WHAT and WHY, not HOW** — no code, no schemas, no implementation details
