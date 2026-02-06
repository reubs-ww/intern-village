---
name: implementation-plan
description: Generate detailed implementation checklists from specification documents. Use when asked to create an implementation plan, impl plan, or implementation checklist for a spec. Transforms architecture/design specs into phased, actionable tasks with file listings, verification steps, and progress tracking. Triggers on requests like "create an implementation plan for this spec", "break down this spec into tasks", or "write an impl.md for this feature".
---

# Implementation Plan

Generate a phased implementation checklist from a specification document. The output is a markdown file that serves as a developer's guide for implementing the spec.

## Workflow

1. **Read the specification** - Understand scope, requirements, and architecture
2. **Identify phases** - Group work into logical, sequential phases
3. **Extract tasks** - Break each phase into checkbox items with file references
4. **Add verification** - Define how to verify each phase is complete
5. **List file changes** - Summarize files to create and modify

## Output Structure

```markdown
# {Feature} Implementation Plan

Implementation checklist for `specs/{feature}.md`. Each item cites the relevant specification section.

---

## Phase N: {Phase Name}

**Reference:** [{spec}.md §X](./{spec}.md#section)

**Status:** 🔲 Not Started | 🔄 In Progress | ✅ COMPLETED

**Goal:** {One sentence describing what this phase achieves}

- [ ] Task description
  - Sub-detail or file path
  - Reference to spec section

**Verification:**
- [ ] Verification step 1
- [ ] Verification step 2

---

## Files to Create

```
path/to/
├── new-file.ts
└── another-file.go
```

## Files to Modify

| File | Change |
|------|--------|
| `path/to/file.ts` | Add X, update Y |

## Verification Checklist

- [ ] Build succeeds
- [ ] Tests pass
- [ ] Feature works end-to-end

## Decisions Made

| Question | Decision |
|----------|----------|
| {Open question from spec} | {Resolution} |
```

## Phase Guidelines

### Phase Sizing
- Each phase should be completable in one focused session
- Aim for 5-15 tasks per phase
- Group by dependency: earlier phases shouldn't depend on later ones

### Phase Ordering
1. **Foundation** - Types, config, project setup
2. **Data layer** - Database, schemas, repositories
3. **Core logic** - Business logic, services
4. **API/Interface** - Handlers, routes, UI components
5. **Integration** - Wiring components together
6. **Polish** - Error handling, edge cases, docs
7. **Testing** - Unit, integration, e2e tests
8. **Deployment** - Docker, CI/CD, infrastructure

### Task Format

Good tasks are:
- **Actionable**: Start with a verb (Create, Add, Update, Implement)
- **Specific**: Reference exact files and functions
- **Traceable**: Link to spec sections with `[spec.md §N](./spec.md#section)`

```markdown
- [ ] Create `src/types/user.ts`
  - `User` interface with id, email, name
  - See [spec.md §4.1](./spec.md#41-user-model)
```

### Status Indicators

Use consistent status markers:
- `🔲 Not Started` or no marker
- `🔄 In Progress`
- `✅ COMPLETED` with completion date/commit if available

## Reference

See [references/template.md](references/template.md) for a full template with all sections.
