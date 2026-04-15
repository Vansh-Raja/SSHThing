# SSHThing Teams: Execution Plan

## Summary

This document is the working execution plan for the Teams feature.
It turns the product decisions from the other Teams docs into a concrete sequence of work.

The order is deliberate:

1. freeze the user experience
2. freeze the role and policy model
3. freeze the backend and security model
4. only then design the coding breakdown

This prevents the engineering plan from getting ahead of unresolved product decisions.

## Current Inputs

The following docs are the current source of truth:

- `TEAMS_FEATURE_PLAN.md`
- `TEAMS_MOCK_SCREENS.md`
- `TEAMS_PERMISSION_MATRIX.md`
- `TEAMS_MVP_SCOPE.md`
- `TEAMS_ARCHITECTURE_PLAN.md`

This file does not replace them.
It sequences them.

## Planning Outcome We Want

The planning stream is complete when we have:

- a frozen Teams vocabulary
- a frozen screen set
- a frozen permission model
- a frozen MVP boundary
- a frozen backend posture for MVP
- a concrete engineering rollout plan with no unresolved product blockers

## Phase 0: Worktree Baseline

### Goal

Keep all Teams planning isolated from `main` while the feature is still being defined.

### Status

- worktree path: `D:\Code\SSHThing-teams`
- branch: `feat/teams-foundation`

### Output

- isolated branch for all Teams planning work
- no runtime feature code required yet

## Phase 1: Product Vocabulary Freeze

### Goal

Lock the names and concepts before the UI or backend drift in different directions.

### Decisions to Freeze

- `Personal` and `Teams` are separate product modes
- `Workspace` is the org boundary
- `Vault` is the scoped collection inside a workspace
- `Resource` is the connectable item inside a vault
- `connect` and `reveal` are different permissions
- MVP supports shared metadata and role-aware visibility
- brokered access is a later capability, not an MVP assumption

### Deliverables

- `TEAMS_FEATURE_PLAN.md` updated and internally consistent

### Exit Criteria

- there is no conflicting terminology across the docs
- personas map cleanly to formal roles
- no screen or policy depends on undefined terms

## Phase 2: Mock-Screen Freeze

### Goal

Decide what the feature looks like in the TUI before deciding how it is built.

### Screens to Freeze

1. `Login / Teams entry`
2. `Workspace switcher`
3. `Vault list`
4. `Admin resource list`
5. `Restricted operator resource list`
6. `Admin resource detail`
7. `Restricted operator resource detail`
8. `Invite/member management`
9. `Role assignment`
10. `Access request / approval flow`

### Required Review Questions Per Screen

- what is the purpose of this screen
- who can reach it
- what information is visible
- what information is masked
- what actions are available
- which actions are disabled with explanation
- which actions are fully hidden
- what keyboard shortcuts exist
- where does navigation go next
- what does the empty state say
- what does the error state say
- what does the loading state say
- what happens on permission failure

### Deliverables

- `TEAMS_MOCK_SCREENS.md` frozen as the UI source of truth

### Exit Criteria

- admin and restricted-operator views are meaningfully different
- no restricted screen leaks hostname, IP, username, or raw secret by accident
- all restricted actions are consistently hidden, disabled, or request-based
- the screen set is enough to explain the MVP without backend assumptions

## Phase 3: Permission Matrix Freeze

### Goal

Turn the screen behavior into a formal policy model.

### Roles in Scope

Workspace level:

- `Owner`
- `Admin`

Vault level:

- `Vault Admin`
- `Editor`
- `Operator`
- `Restricted Operator`
- `Viewer`
- `Requester`

### Capability Set to Freeze

- see workspace
- see vault
- see resource row
- see hostname/IP
- see username
- see notes/tags
- connect
- reveal password/key
- copy/export credentials
- edit resource
- invite members
- remove members
- change roles
- approve temporary access
- see audit trail

### Persona Mapping to Lock

- `Intern` -> `Restricted Operator`
- `Full-time Operator` -> `Operator`
- `Manager` -> `Vault Admin` or `Editor`
- `Platform Admin` -> `Workspace Admin`

### Deliverables

- `TEAMS_PERMISSION_MATRIX.md` frozen

### Exit Criteria

- every role has a clear answer for every capability
- the intern scenario is fully represented
- offboarding implications are documented
- the permission matrix matches the screen behavior exactly

## Phase 4: MVP Boundary Freeze

### Goal

Prevent the first implementation pass from collapsing under too much scope.

### MVP Must Include

- Teams sign-in entry
- workspace list and switching
- vault list and selection
- shared resource browsing
- admin resource detail
- restricted-operator resource detail
- invite and membership listing
- role assignment
- connect vs reveal separation in the UI model

### MVP Must Exclude

- brokered sessions
- ephemeral credentials
- session recording
- enterprise SSO/SCIM
- self-hosted deployment UX
- GitHub-backed team storage
- GH-driven onboarding
- temporary access grants as a complete workflow

### Deliverables

- `TEAMS_MVP_SCOPE.md` frozen

### Exit Criteria

- the MVP can be described in one short paragraph
- anything not required for the first usable Teams flow is explicitly deferred
- no screen depends on a post-MVP feature to make sense

## Phase 5: Backend and Security Architecture Freeze

### Goal

Choose the canonical architecture for MVP so the coding plan targets one system, not three.

### MVP Backend Direction

- canonical product: `SSHThing Cloud Teams`
- future variant: `SSHThing Teams Self-Hosted`
- future advanced option: `Git-backed team storage`

### Architecture Topics to Freeze

#### Identity Model

- user account
- auth session
- device identity
- workspace membership

#### Shared Data Model

- workspace
- vault
- resource
- notes/snippets
- access policy
- membership and roles

#### Secret Ownership Model

- personal mode keeps existing password-based local DB
- Teams mode uses a separate team-owned encryption domain
- vault access is member-specific and revocable
- removing a member may require secret rotation if raw secret access existed

#### Access Models

- personal credential on shared host
- shared team credential
- future brokered access

#### Audit Model

- membership change events
- role change events
- resource change events
- connection launch events
- secret reveal events
- access request events

### GitHub / GH Position

- not the source of truth for Teams permissions
- not an MVP dependency
- possible future setup helper or advanced backend option

### Convex Position

- plausible implementation substrate
- should remain an internal technical choice
- should not define the user-facing product language

### Deliverables

- `TEAMS_ARCHITECTURE_PLAN.md` frozen

### Exit Criteria

- Personal and Teams coexistence rules are clear
- no architecture assumption contradicts the permission matrix
- the backend posture for MVP is singular and unambiguous

## Phase 6: Engineering Breakdown

### Goal

Turn the frozen product design into a coding plan with clear tracks and dependencies.

### Track A: Teams Navigation Shell

Scope:

- Teams entry point alongside Personal mode
- top-level navigation updates
- workspace and vault selection shell

Depends on:

- product vocabulary freeze
- mock-screen freeze

### Track B: Teams Domain Model

Scope:

- client-side Teams entities and view models
- shared vocabulary used across screens and backend boundaries

Depends on:

- product vocabulary freeze
- permission matrix freeze
- architecture freeze

### Track C: Authentication and Session Bootstrap

Scope:

- user login flow
- session restore
- workspace membership bootstrap
- device/session handling assumptions for Teams mode

Depends on:

- architecture freeze
- MVP scope freeze

### Track D: Vault and Resource Browsing

Scope:

- workspace switcher
- vault list
- admin resource list
- restricted operator resource list
- role-aware detail payload assumptions

Depends on:

- mock-screen freeze
- permission matrix freeze
- auth/session foundation

### Track E: Membership and Role Management

Scope:

- invite flow
- member list
- role assignment UI
- remove member flow

Depends on:

- permission matrix freeze
- architecture freeze
- auth/session foundation

### Track F: Connect and Secret Policy

Scope:

- connect-capable actions in Teams mode
- policy-aware reveal/export rules
- restricted-operator behavior
- shared-credential and personal-credential resource handling

Depends on:

- permission matrix freeze
- architecture freeze
- vault/resource browsing

### Track G: Audit and Access Request Foundations

Scope:

- audit event model
- initial request-access UI scaffolding
- request visibility and approval boundaries

Depends on:

- permission matrix freeze
- architecture freeze
- membership model

## Phase 7: Dependency Order for Actual Coding

When coding starts, the recommended order is:

1. Teams navigation shell
2. Teams domain model
3. auth and session bootstrap
4. workspace and vault browsing
5. role-aware resource list and detail screens
6. membership and role management
7. connect and secret policy enforcement
8. audit and access-request foundations

This order keeps the app usable at each layer instead of jumping straight into secret handling.

## Phase 8: Delivery Milestones

### Milestone 1: Design Complete

Outputs:

- all planning docs frozen
- no unresolved role or screen contradictions

### Milestone 2: Read-Only Teams Prototype

Outputs:

- Teams mode entry
- workspace and vault browsing
- role-aware resource rendering

### Milestone 3: Manageable Teams Prototype

Outputs:

- member list
- invites
- role assignment
- restricted-operator behavior visible in the UI

### Milestone 4: Connect-Capable MVP

Outputs:

- users can sign in to Teams mode
- select a workspace and vault
- browse resources according to role
- connect where policy allows
- reveal/export remains role-gated

### Milestone 5: Governance Follow-Up

Outputs:

- audit feed foundation
- request-access model
- post-MVP planning for brokered access and stronger controls

## Review Checkpoints

Before coding starts, review the following in order:

1. mock screens
2. permission matrix
3. MVP scope
4. architecture plan
5. execution plan

If any earlier document changes, review this plan again and update dependencies.

## Risks to Watch Early

- overloading MVP with brokered-access ideas too early
- letting GitHub integration distort the core product model
- making restricted screens too weak or too confusing
- mixing Personal-mode encryption assumptions into Teams mode
- treating reveal/export as equivalent to connect
- designing offboarding without accounting for shared-secret rotation

## Acceptance Criteria

This execution plan is ready to hand off into real engineering planning when:

- the Teams mock screens are approved
- the role model is approved
- the MVP line is approved
- the backend posture is approved
- the work can be split into coding tracks without unresolved product questions

At that point, the next document should be a code-oriented implementation plan tied to the existing Go codebase.
