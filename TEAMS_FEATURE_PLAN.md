# SSHThing Teams: Product + Technical Brainstorm

## Summary

SSHThing Teams should not be treated as "Git sync, but for more people."
It should be treated as a separate product layer on top of SSHThing Personal:

- `Personal`: local-first encrypted database, private sync across a single user's devices
- `Teams`: account-based workspaces, shared vaults/projects, membership, roles, access policies, and optional backend choices

The most important product distinction is this:

- "Can connect" must be separable from
- "Can view host metadata" and
- "Can reveal/export raw credentials"

That is the core capability that enables intern-style restricted access while still allowing trusted employees to manage systems properly.

---

## Product Direction

### What Teams Should Mean

SSHThing Teams should introduce these concepts:

- `Account`
- `Workspace`
- `Vault` or `Project`
- `Member`
- `Role`
- `Resource`
- `Credential policy`
- `Access policy`

A workspace is the team/company boundary.
A vault/project is a scoped area such as:

- `Production`
- `Staging`
- `Client A`
- `Internal Tools`

This avoids one giant flat list of shared servers and makes permissions manageable.

### Recommended Product Positioning

- `SSHThing Personal`: current product
- `SSHThing Teams`: premium/cloud-backed collaborative mode
- `SSHThing Teams Self-Hosted`: enterprise/self-hosted deployment option

### Backend Positioning

Users should not think in terms of "bring your own Convex database."
They should think in terms of:

- `SSHThing Cloud`
- `SSHThing Self-Hosted`
- optional `Git-backed/self-managed team storage` for advanced users later

Convex can still be the implementation substrate for the managed and self-hosted team backend.

---

## Core Feature Model

### Access Modes Per Resource

Each shared resource should support one of these models:

1. `Shared host, personal credentials`
   - Team sees the host
   - Each user provides their own SSH identity or is mapped to their own account
   - Strongest default for many companies

2. `Shared host, shared credentials`
   - Team shares raw secret material
   - Fastest to ship
   - Weakest offboarding story because users can copy secrets

3. `Brokered access`
   - Team members can connect without receiving the long-lived raw credential
   - Best long-term security model
   - Best fit for "intern can connect but cannot reveal anything"

### Recommended Default

Default to:

- shared host metadata
- personal credentials where possible
- shared credentials only when explicitly chosen by admins

Treat brokered access as the long-term differentiator.

---

## Roles and Permissions

### Workspace-Level Roles

- `Owner`
- `Admin`

### Vault-Level Roles

- `Vault Admin`
- `Editor`
- `Operator`
- `Restricted Operator`
- `Viewer`
- `Requester`

### Permission Dimensions

Permissions must be split across:

- `discover_resource`
- `view_metadata`
- `connect`
- `reveal_secret`
- `copy_export_secret`
- `edit_resource`
- `manage_membership`
- `approve_access`

### Example Role Profiles

#### Restricted Operator (intern)

- can discover assigned resources
- can connect to approved resources
- cannot reveal password/private key
- cannot export credentials
- may see masked metadata
- may have hidden hostname/IP depending on policy

#### Operator

- can connect
- can view standard metadata
- may reveal secrets if vault policy allows

#### Vault Admin

- can create/edit/delete resources
- can manage role assignments in that vault
- can choose credential mode
- can reveal or rotate shared credentials

---

## Visibility Policy

The app should support different visibility levels for a resource:

- `Full`
  - show label, hostname/IP, username, port, notes, tags

- `Masked`
  - show human label and environment info
  - hide IP/hostname/username

- `Connect-only`
  - resource is visible enough to launch a session
  - almost all raw details hidden

This is required for the "intern can connect but can't see the infrastructure details" use case.

---

## Encryption and Secret Ownership

### Current SSHThing Model

Today SSHThing Personal is password-encrypted and local-first.
That is good for personal mode, but it is not sufficient for team sharing.

### Teams Model

Team data should not be encrypted with any one user's personal login password.

Instead:

- each team vault should have its own encryption context
- vault contents should be encrypted with a vault-level key
- the vault key should be wrapped separately for each authorized member
- member removal should remove access to future vault state

### Important Product Truth

If users ever receive raw shared credentials, removing them from the team does not retroactively protect those credentials.

Therefore Teams must eventually support:

- member removal
- access revocation
- "credential rotation required" workflows
- brokered/non-exportable access as a stronger model

---

## Accounts and Identity

### Recommendation

Teams mode should require account-based identity.

Without accounts, SSHThing cannot properly support:

- invites
- offboarding
- per-user roles
- approvals
- audit history
- device trust
- temporary access grants

### Product Split

- Personal mode can remain account-optional
- Teams mode should be account-backed

---

## Convex Fit

Conceptually, Convex is a good fit for:

- authenticated users
- shared workspaces and vaults
- realtime updates
- role-aware queries
- managed cloud and self-hosted modes

So the backend direction can be:

- `SSHThing Cloud Teams` powered by Convex
- `SSHThing Self-Hosted Teams` powered by self-hosted Convex or an equivalent internal deployment

But users should experience this as "SSHThing Teams", not "a random Convex DB hookup."

---

## GitHub / GH Integration

### Where GitHub Fits

GitHub should be treated as an integration and optional setup convenience, not the core permission model.

Good uses:

- bootstrap a self-managed team repo
- select an org/repo during setup
- import candidate collaborators/org members
- simplify auth flows through `gh`

### Where GitHub Does Not Fit as the Core Model

GitHub repository access does not solve:

- connect without reveal
- per-vault resource visibility
- temporary access requests
- app-native audit trails
- resource-level permissions

### Recommendation

If GitHub is used:

- use it as a setup and optional backend-storage helper
- do not use GitHub membership as the only source of SSHThing authorization

---

## TUI Product Design Direction

This feature needs its own first-class navigation structure.

### Top-Level Navigation

- `Personal`
- `Teams`
- `Access Requests`
- `Audit`
- `Settings`

### Teams Area

Inside `Teams`, users should be able to:

- switch workspace
- switch vault/project
- browse resources
- inspect details subject to role
- manage members if permitted
- review invitations and pending requests

### Resource List Behavior

Different users should see different row detail levels.

#### Admin/Full Operator

- `prod-api-1`
- `10.42.3.18`
- `ubuntu`
- `Shared key`
- `Vault: Production`

#### Restricted Operator / Intern

- `Production API Node`
- `Production`
- `Connect allowed`
- `Address hidden`
- `Credential hidden`

### Important UI Principle

Do not merely disable actions silently.
Explain why.

Examples:

- `Reveal credential: not permitted by vault policy`
- `Host address hidden for your role`
- `Request elevated access to reveal raw details`

---

## Why Mock Screens Should Come First

This is the correct next step.

The UI is not just presentation here. It determines:

- what concepts are first-class
- how visibility restrictions are expressed
- how teams, vaults, and roles are organized
- what "connect without reveal" actually looks like
- how much workflow complexity the user can tolerate

If we skip screens and go straight to backend design, we risk building the wrong abstractions.

Therefore the work should proceed in this order:

1. finalize product concepts
2. finalize mock screens and user flows
3. lock role/visibility rules
4. then design technical architecture

---

## Technical Brainstorm (No Code)

### Recommended Overall Architecture

Use a dual-mode product model:

- `Local personal mode`
  - existing SSHThing DB remains the source of truth

- `Account-backed teams mode`
  - workspace/vault data comes from a service backend
  - local cache may exist for UX, but authority is the workspace backend

### Logical Objects

- `User`
- `Device`
- `Workspace`
- `Vault`
- `Membership`
- `RoleAssignment`
- `Resource`
- `ConnectionPolicy`
- `CredentialEnvelope`
- `AccessGrant`
- `AccessRequest`
- `AuditEvent`

### Resource Concept

Do not model this only as "a host row."
Model it as a resource/target that can be connected to under policy.

This keeps the door open for:

- SSH hosts
- SFTP targets
- snippets/runbooks
- future services beyond SSH

### Credential Strategy

Support all three conceptual models:

- `personal credential attached to shared host`
- `shared team credential`
- `brokered access`

This should be a property of the resource policy, not a global mode.

### Device Model

Teams mode likely needs some notion of trusted device state:

- approved device
- new device pending approval
- device lost / revoked

This becomes important once access is no longer just personal-local.

### Audit Requirements

Minimum useful audit events:

- invite sent
- member joined
- member removed
- role changed
- resource created/edited/deleted
- credential revealed
- connection launched
- access requested
- access approved/denied

### Offboarding Requirements

Teams mode must explicitly support:

- remove member
- invalidate future vault access
- mark affected shared credentials for rotation
- track whether rotation has happened

---

## Recommended MVP Scope

### MVP

- account login
- workspace creation
- vault/project structure
- invites and membership
- per-vault roles
- shared resources
- restricted operator mode
- optional secret reveal based on role
- connect from TUI based on role and policy

### Post-MVP

- access requests
- temporary grants
- audit feed
- GitHub-assisted team bootstrap
- self-hosted backend option

### Long-Term

- brokered sessions
- ephemeral credentials or certificates
- stronger anti-export guarantees
- enterprise SSO/SCIM

---

## Immediate Planning Recommendation

Before implementation planning, finalize these mock screens:

1. `Login / workspace entry`
2. `Workspace switcher`
3. `Vault list`
4. `Resource list for admin`
5. `Resource list for restricted operator`
6. `Resource detail panel for admin`
7. `Resource detail panel for restricted operator`
8. `Invite/member management`
9. `Role assignment flow`
10. `Access request flow`

Only after those screens are agreed should the technical design be frozen.

---

## Next-Step Plan

### Phase 0: Mock Screens and Product Freeze

- define top-level Teams information architecture
- define screen list and navigation
- define exact role names and permission semantics
- define visibility rules for restricted roles
- define which actions appear, hide, or explain themselves
- define the first-run setup experience for teams mode

### Phase 1: Domain and Backend Design

- finalize object model
- finalize encryption/key ownership model
- finalize cloud vs self-hosted backend posture
- finalize how GitHub integration fits, if at all
- finalize how personal mode and teams mode coexist

### Phase 2: Interaction and Policy Design

- member invite flows
- member removal flows
- credential sharing modes
- connect vs reveal behaviors
- request/approval workflows
- audit and offboarding expectations

### Phase 3: Implementation Planning

- translate frozen screens and policies into backend and client workstreams
- break down migration risks
- define rollout sequencing
- define what ships in MVP versus later phases

