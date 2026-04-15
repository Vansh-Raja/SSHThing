# SSHThing Teams: Permission Matrix

## Summary

This document freezes the role model for SSHThing Teams MVP.
The key principle is that these capabilities remain distinct:

- discover
- view metadata
- connect
- reveal
- export
- edit
- manage members
- approve access

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

## Persona Mapping

| Persona | Default Role |
|---|---|
| Intern | Restricted Operator |
| Full-time Operator | Operator |
| Manager | Vault Admin |
| Platform Admin | Workspace Admin |

## Capability Definitions

| Capability | Meaning |
|---|---|
| See workspace | User can see workspace in switcher |
| See vault | User can see vault in workspace |
| See resource row | User can see resource listed |
| See hostname/IP | User can see real hostname or address |
| See username | User can see login username |
| See notes/tags | User can see operational notes and tags |
| Connect | User can launch connection |
| Reveal password/key | User can display raw secret material |
| Copy/export credentials | User can copy or export raw secret material |
| Edit resource | User can create or modify resources |
| Invite members | User can send invitations |
| Remove members | User can remove members from vault/workspace |
| Change roles | User can change role assignments |
| Approve temporary access | User can approve access requests |
| See audit trail | User can inspect audit history |

## MVP Permission Matrix

| Role | See workspace | See vault | See resource row | See hostname/IP | See username | See notes/tags | Connect | Reveal password/key | Copy/export credentials | Edit resource | Invite members | Remove members | Change roles | Approve temporary access | See audit trail |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Owner | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Admin | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Vault Admin | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Policy | Policy | Yes | Yes | Yes | Yes | Yes | Yes |
| Editor | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Policy | Policy | Yes | No | No | No | No | Limited |
| Operator | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Policy | Policy | No | No | No | No | No | Limited |
| Restricted Operator | Yes | Yes | Yes | No by default | No by default | Limited | Yes if granted | No | No | No | No | No | No | No | No |
| Viewer | Yes | Yes | Yes | Policy | Policy | Policy | No | No | No | No | No | No | No | No | Limited |
| Requester | Yes | Yes | Limited | No | No | No | No by default | No | No | No | No | No | No | No | No |

## Policy Overrides

### Policy-Governed Fields

The matrix uses `Policy` where the vault or resource policy may allow or deny the capability.
For MVP, the following are policy-governed:

- reveal password/key
- copy/export credentials
- some metadata visibility for `Viewer`
- some metadata visibility for `Restricted Operator`

### Default Policy for MVP

Use these defaults unless a vault explicitly changes them:

- `Vault Admin`
  - reveal allowed
  - export allowed

- `Editor`
  - reveal disallowed by default
  - export disallowed by default

- `Operator`
  - reveal disallowed by default
  - export disallowed by default

- `Restricted Operator`
  - reveal never allowed in MVP
  - export never allowed in MVP

- `Viewer`
  - no connect
  - no reveal
  - no export

- `Requester`
  - can discover requestable resources only
  - must use access request flow

## Metadata Visibility Rules

### Owner / Admin / Vault Admin / Operator

Default metadata visibility:

- hostname/IP visible
- username visible
- notes/tags visible

### Restricted Operator

Default metadata visibility:

- resource label visible
- environment visible
- hostname/IP hidden
- username hidden
- low-sensitivity notes only

### Viewer

Vault policy decides whether this role sees:

- full metadata
- masked metadata
- only friendly names

### Requester

Requester should only see enough to understand what they may request.
Do not expose hidden infrastructure details through the request flow.

## Rendering Rules by Permission Outcome

### Connect

- if allowed: visible action
- if requestable: visible as `request access`
- if not allowed and not requestable: disabled with explanation or hidden based on screen spec

### Reveal / Export

- hidden for `Restricted Operator` and `Requester`
- disabled with explanation for `Operator` when vault policy disallows it
- visible for admin roles when allowed

### Edit / Membership / Role Management

- hidden from non-admin operational roles
- visible to vault admins and above

## Audit Visibility

### MVP Recommendation

- Owner/Admin/Vault Admin: full audit visibility within scope
- Editor/Operator: limited audit summary only
- Restricted Operator/Requester: no audit feed by default

### Limited Audit Summary

Limited summary means:

- recent access decisions affecting the current user
- recent changes relevant to the current vault
- no broad organization-wide audit visibility

## Offboarding Rules

When a user is removed:

- they immediately lose future workspace/vault access
- if they ever had raw credential reveal/export rights, affected resources should be marked `rotation required`

This is a product rule and must be reflected in future audit/offboarding design.
