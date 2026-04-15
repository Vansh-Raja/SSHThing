# SSHThing Teams: Architecture and Implementation Planning

## Summary

This document turns the Teams product model into a technical design direction without going into code-level implementation.

The architecture is built around these principles:

- Personal and Teams are separate authority domains
- Teams is account-backed
- shared data belongs to workspaces and vaults, not to one member's personal password
- connect, reveal, and export are separate permission outcomes
- the client should support future hosted and self-hosted variants

## Recommended MVP Architecture Direction

### Canonical MVP Product

- `SSHThing Cloud Teams`

### Future Variants

- `SSHThing Teams Self-Hosted`
- `Advanced: Git-backed team storage`

### Why This Direction

Designing against a service-backed Teams model first makes it much easier to define:

- identity
- invites
- role assignments
- visibility-filtered resource payloads
- access lifecycle
- audit events

Git-backed storage remains valuable, but should not dictate the core product architecture.

## Domain Model

### Core Objects

- `User`
- `Device`
- `Workspace`
- `Vault`
- `Membership`
- `RoleAssignment`
- `Resource`
- `VisibilityPolicy`
- `CredentialMode`
- `AccessRequest`
- `AuditEvent`

### Object Intent

#### User

- account identity for Teams mode

#### Device

- identifies a local app installation or approved client device later

#### Workspace

- top-level team boundary

#### Vault

- scoped project inside a workspace

#### Membership

- user membership in workspace or vault context

#### RoleAssignment

- formal permission assignment for workspace or vault scope

#### Resource

- shared host or adjacent object inside a vault
- MVP assumes SSH-oriented resources first

#### VisibilityPolicy

- controls what fields are shown to which roles

#### CredentialMode

- personal credentials
- shared credentials
- future brokered access

#### AccessRequest

- future mechanism for requesting additional rights

#### AuditEvent

- immutable record of important actions

## Identity Model

### MVP Identity Assumptions

- Teams mode requires authenticated users
- Personal mode remains account-optional

### Minimum Teams Identity Capabilities

- sign in
- list workspaces
- resolve memberships
- load role-specific data

### Device Model

Device trust is not an MVP blocker, but the model should leave room for:

- known devices
- revoked devices
- approval-required devices later

## Shared Data Model

### Workspace Data

- name
- description
- owner/admin memberships
- vault list

### Vault Data

- name
- description
- role assignments
- resource list
- policy toggles

### Resource Data

- friendly label
- connection details
- metadata fields
- notes/tags
- credential mode
- policy flags

## Secret Ownership Model

### Personal Mode

- current local encrypted DB remains authoritative

### Teams Mode

- workspace/vault secrets must not be derived from one member's personal DB password
- vault data should have its own encryption context
- authorized members should receive access to vault data without requiring a shared human password

### Product-Level Requirement

Removing a member must revoke future vault access.
If that member previously had raw secret reveal rights, the system must mark impacted credentials as requiring rotation.

## Access Models

### 1. Personal Credential on Shared Host

- team shares the host entry
- each member attaches or maps their own credential
- best default posture

### 2. Shared Team Credential

- vault stores shared secret material
- some roles may reveal/export it
- restricted roles may only connect, if product policy permits

### 3. Future Brokered Access

- raw credential never needs to be directly exposed to end users
- best long-term model for restricted operator use cases

## Visibility Filtering

The backend must be able to serve role-aware payloads.

Examples:

- admin payload includes hostname/IP, username, and secret controls
- restricted payload includes only friendly label, environment, and allowed actions

This is preferable to sending full data and hiding fields purely in the client.

## Audit Model

### Minimum Audit Events

- invite sent
- invite accepted
- member removed
- role changed
- resource created
- resource updated
- credential revealed
- connection launched
- access request created
- access request resolved

### MVP vs Later

- full audit UI can be post-MVP
- audit event generation should still exist in the model

## Offboarding Model

### Must Support

- remove member
- revoke future vault access
- identify resources that need rotation
- show `rotation required` state

### Not Required for MVP

- automated credential rotation
- ephemeral certificate replacement

## GitHub / GH Integration Boundary

### MVP Position

GitHub is not a core dependency of the Teams architecture.

### Future Integration Use Cases

- detect `gh`
- list repos and orgs
- help bootstrap advanced Git-backed teams storage
- import org membership candidates

### Rule

GitHub integration must not become the source of truth for Teams permissions.

## Convex Positioning

Convex is a suitable implementation substrate for:

- auth-backed app state
- workspace/vault data
- realtime membership and resource updates
- hosted and self-hosted deployment models

It should remain an implementation detail.

## Client-Side State Planning

### Current App Structure

The current TUI has:

- `PageHome`
- `PageSettings`
- `PageTokens`

### Teams Expansion

The client state model will need future pages or top-level modes for:

- Personal
- Teams
- Access Requests
- Audit

### Teams State Categories

- active mode
- signed-in account state
- selected workspace
- selected vault
- selected resource
- current role context
- resource detail visibility level

## Backend Boundary Planning

The eventual client/backend contract must support:

- sign-in bootstrap
- workspace list
- vault list
- resource list filtered by role
- resource detail filtered by role
- member list and role assignments
- invitations
- access request creation and resolution
- audit retrieval

## Coexistence with Personal Mode

### Rule

Personal and Teams must coexist without corrupting each other.

### Required Product Behavior

- personal local DB remains untouched
- Teams data is not authored in the personal DB as the source of truth
- local Teams cache may exist for UX later, but is not authoritative
- existing Git sync remains Personal-only until explicitly redesigned

## Delivery Sequence

This is the step-by-step sequence after the design docs are frozen.

### Phase 0: Design Freeze

- finalize mock screens
- finalize permission matrix
- finalize MVP scope
- finalize product vocabulary

### Phase 1: Domain Freeze

- freeze the data model
- freeze role semantics
- freeze visibility policies
- freeze credential mode semantics

### Phase 2: Client UX Planning

- map existing TUI to Teams navigation
- define new pages and overlays
- define search and detail behavior under restricted visibility

### Phase 3: Backend Contract Planning

- define requests and responses required by the client
- define filtered payload strategy
- define membership and audit boundaries

### Phase 4: Security and Access Flow Planning

- define personal credential flow
- define shared credential flow
- define future brokered flow
- define offboarding effects

### Phase 5: Rollout Planning

- internal mock review
- hidden prototype
- read-only Teams browsing
- connect-capable MVP
- request/governance follow-up

## Immediate Next Step

The next step after this document is not coding.
It is to review `TEAMS_MOCK_SCREENS.md`, `TEAMS_PERMISSION_MATRIX.md`, and `TEAMS_MVP_SCOPE.md` together and remove contradictions before creating the final engineering implementation plan.
