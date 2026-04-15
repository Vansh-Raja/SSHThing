# SSHThing Teams: Feature Plan

## Summary

SSHThing Teams should be a separate product mode from SSHThing Personal.
It should introduce account-backed collaboration, shared workspaces, vaults/projects, team membership, roles, and access policies that allow a user to connect without necessarily revealing raw infrastructure details or credential material.

This worktree exists to design the feature end-to-end before implementation.
The design order is intentional:

1. freeze the product model
2. freeze the TUI mock screens
3. freeze the permission matrix
4. freeze the MVP boundary
5. freeze the backend and encryption architecture
6. only then create the implementation-ready engineering plan

## Product Positioning

### Product Modes

- `SSHThing Personal`
  - current local-first mode
  - local encrypted DB
  - optional Git sync across the same user's devices

- `SSHThing Teams`
  - account-backed collaborative mode
  - shared workspaces and vaults
  - role-aware access and visibility rules
  - optional future self-hosted deployment

### Why Teams Must Be Separate

The current personal sync model assumes a single user's encryption domain and is designed for device sync, not team collaboration.
Teams must not inherit that authority model.
Shared team data should belong to the team/workspace, not to one member's personal password.

## Core Product Model

### Hierarchy

- `User`
- `Workspace`
- `Vault`
- `Resource`

### Definitions

- `User`
  - an authenticated account in Teams mode

- `Workspace`
  - the org/team boundary
  - owns members, billing, global admins, and vaults

- `Vault`
  - a scoped collection inside a workspace
  - examples: `Production`, `Staging`, `Client A`, `Internal Tools`

- `Resource`
  - a connectable or visible item inside a vault
  - MVP assumes SSH/SFTP targets first

## Product Principles

### Access Must Be Separated

These are separate permissions:

- discover resource
- view metadata
- connect
- reveal secret
- copy/export secret
- edit resource
- manage members
- approve access

The feature must never collapse `connect` into `reveal`.

### Shared Access Modes

Each resource should support one of these conceptual access modes:

1. `Shared host, personal credentials`
2. `Shared host, shared credentials`
3. `Brokered access`

### Recommended Default Posture

For MVP:

- shared host metadata is supported
- personal credentials are preferred where possible
- shared credentials are allowed explicitly
- brokered access is planned but not required for MVP

## Roles

### Workspace Roles

- `Owner`
- `Admin`

### Vault Roles

- `Vault Admin`
- `Editor`
- `Operator`
- `Restricted Operator`
- `Viewer`
- `Requester`

### Intended Persona Mapping

- `Intern` -> `Restricted Operator`
- `Full-time Operator` -> `Operator`
- `Manager` -> `Vault Admin` or `Editor` depending on need
- `Platform Admin` -> `Workspace Admin` or `Owner`

## Visibility Model

Resources must support policy-driven visibility:

- `Full`
  - show hostname/IP, username, notes, tags, and credential metadata

- `Masked`
  - show friendly name and environment context
  - hide hostname/IP and raw connection details

- `Connect-only`
  - allow launch of approved sessions
  - do not reveal sensitive resource details

This is the basis of the intern-style flow where the user can connect but not inspect infrastructure details.

## Teams Mode Requirements

### Required for MVP

- account login
- workspace creation and switching
- vault list and resource list
- invite and member management
- role assignments
- restricted operator mode
- connect vs reveal separation
- admin/operator/restricted-operator screen differences

### Required After MVP

- access requests
- temporary access grants
- audit feed
- GitHub-assisted backend bootstrap
- self-hosted teams backend

### Long-Term

- brokered sessions
- ephemeral credentials or certificates
- stronger non-exportability guarantees
- enterprise SSO/SCIM

## Backend Direction

### MVP Recommendation

Treat `SSHThing Cloud Teams` as the canonical MVP model.

Future/advanced variants:

- `SSHThing Teams Self-Hosted`
- `Advanced: Git-backed team storage`

### Convex Positioning

Convex is a plausible implementation substrate for both hosted and self-hosted Teams variants.
It should remain an internal backend decision, not a direct user-facing concept.

Users should see:

- `SSHThing Cloud Teams`
- `SSHThing Teams Self-Hosted`

They should not be asked to think in terms of "bring your own random Convex DB".

## GitHub / GH Integration Position

GitHub is useful as:

- a setup helper
- an org/repo picker
- a future advanced backend option

GitHub is not sufficient as the source of truth for:

- per-resource visibility
- connect vs reveal permissioning
- access requests
- audit and offboarding workflows

MVP should not depend on GitHub integration.

## TUI Strategy

The Teams feature must become first-class in the application structure.

Recommended top-level navigation expansion:

- `Personal`
- `Teams`
- `Access Requests`
- `Audit`
- `Settings`

The mock screens will define how this fits into the existing TUI.

## Worktree Deliverables

This worktree should contain and maintain these design docs:

- `TEAMS_FEATURE_PLAN.md`
- `TEAMS_MOCK_SCREENS.md`
- `TEAMS_PERMISSION_MATRIX.md`
- `TEAMS_ARCHITECTURE_PLAN.md`
- `TEAMS_MVP_SCOPE.md`
- `TEAMS_IMPLEMENTATION_PLAN.md`

## Design Freeze Order

1. mock screens
2. permission matrix
3. MVP scope
4. architecture plan
5. implementation-ready engineering plan

No runtime code should be written until those are internally consistent.
