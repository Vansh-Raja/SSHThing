# SSHThing Teams: Mock Screen Spec

## Summary

This document defines the first-pass TUI screen model for SSHThing Teams.
Each screen spec includes:

- purpose
- entry points
- visible fields
- hidden or masked fields by role
- actions
- disabled actions and explanation policy
- keybindings
- navigation destinations
- empty state
- error state
- loading state
- permission failure state

## Global Rendering Rules

### Restricted Actions

For any action the current user cannot perform, the screen must explicitly choose one of:

- hidden
- visible but disabled with explanation
- visible as `request access`

### Default Rule

- if user awareness of the capability is useful, show it disabled with explanation
- if user awareness would itself leak sensitive capability details, hide it

### Metadata Visibility

The same resource may render differently for different roles.
The screen spec below assumes this distinction is a core product behavior, not an edge case.

---

## 1. Login / Teams Entry

### Purpose

Let a user choose between Personal mode and Teams mode, then sign into Teams mode.

### Entry Points

- app startup
- command from Personal home
- logout from Teams mode

### Visible Fields

- mode chooser: `Personal` / `Teams`
- if Teams selected:
  - email or identity provider entry point
  - sign-in action
  - environment selector if multiple deployments exist later

### Hidden or Masked by Role

- none

### Actions

- enter Personal mode
- sign into Teams
- switch environment later if supported

### Disabled Actions

- if Teams backend unavailable: show `Teams unavailable` with explanation

### Keybindings

- `↑/↓` move focus
- `Enter` activate selected action
- `Esc` quit app

### Navigation Destinations

- Personal home
- Teams workspace switcher

### Empty State

- not applicable

### Error State

- sign-in failed
- backend unreachable
- unsupported Teams deployment

### Loading State

- signing in
- loading workspaces

### Permission Failure State

- account exists but has no accessible workspaces

---

## 2. Workspace Switcher

### Purpose

Let a Teams user pick which workspace they want to enter.

### Entry Points

- post-login
- workspace switch command from any Teams page

### Visible Fields

- workspace name
- role in workspace
- member count
- badge for current workspace

### Hidden or Masked by Role

- internal billing or owner-only metadata hidden for non-admins

### Actions

- enter workspace
- create workspace if allowed
- leave workspace if allowed later

### Disabled Actions

- create workspace disabled for restricted accounts if policy requires it

### Keybindings

- `↑/↓` navigate
- `Enter` open workspace
- `N` create workspace
- `Esc` go back

### Navigation Destinations

- vault list
- workspace creation flow

### Empty State

- no workspaces yet
- show `Create workspace` and `Join via invite`

### Error State

- failed to load workspace list

### Loading State

- loading workspace memberships

### Permission Failure State

- account signed in but not yet invited anywhere

---

## 3. Vault List

### Purpose

Show all vaults/projects inside the current workspace.

### Entry Points

- workspace switcher
- back from resource list

### Visible Fields

- vault name
- vault description
- role in vault
- resource count
- indicators such as `restricted`, `shared credentials`, `requests enabled`

### Hidden or Masked by Role

- member-only operational notes hidden for low-privilege users

### Actions

- open vault
- create vault if permitted
- manage vault settings if permitted

### Disabled Actions

- create/manage disabled with explanation if user lacks admin rights

### Keybindings

- `↑/↓` navigate
- `Enter` open vault
- `N` create vault
- `M` manage members if allowed
- `Esc` back to workspace switcher

### Navigation Destinations

- admin resource list
- restricted operator resource list
- vault create/edit

### Empty State

- no vaults in workspace
- show `Create vault` if admin, otherwise explanatory message

### Error State

- failed to load vaults

### Loading State

- loading vaults

### Permission Failure State

- workspace visible but no vaults accessible to current role

---

## 4. Admin Resource List

### Purpose

Primary vault view for users who can see full metadata.

### Entry Points

- vault list
- resource detail back navigation

### Visible Fields

- resource label
- hostname or IP
- username
- environment or tags
- credential mode
- status badges

### Hidden or Masked by Role

- not applicable for this screen; only admins/operators with full metadata enter here

### Actions

- connect
- view details
- edit resource
- create resource
- manage vault members
- reveal credential if policy allows

### Disabled Actions

- reveal/export disabled if vault policy forbids it
- member management disabled for non-admin operators

### Keybindings

- `↑/↓` navigate
- `Enter` connect
- `h` open detail
- `a` add resource
- `e` edit resource
- `m` manage members
- `r` reveal secret if allowed
- `Esc` back to vault list

### Navigation Destinations

- admin resource detail
- member management
- add/edit resource

### Empty State

- no resources in vault

### Error State

- failed to load resource list

### Loading State

- loading resources

### Permission Failure State

- if user no longer has full metadata permission, redirect to restricted resource list

---

## 5. Restricted Operator Resource List

### Purpose

Primary vault view for users who may connect but cannot see sensitive metadata.

### Entry Points

- vault list
- restricted resource detail back navigation

### Visible Fields

- friendly resource label
- environment or service category
- allowed actions summary such as `connect allowed`
- request badge such as `elevation available`

### Hidden or Masked by Role

- hostname/IP
- username
- raw credential indicators that would leak too much
- sensitive admin notes

### Actions

- connect if allowed
- request access or elevation
- open restricted detail view

### Disabled Actions

- reveal/export actions should be hidden by default
- edit/manage actions hidden
- connect disabled only if a request flow exists and current grant is missing

### Keybindings

- `↑/↓` navigate
- `Enter` connect or open connect confirmation
- `r` request access
- `h` open detail
- `Esc` back to vault list

### Navigation Destinations

- restricted operator detail
- access request flow

### Empty State

- no accessible resources in this vault

### Error State

- failed to load restricted resource list

### Loading State

- loading permitted resources

### Permission Failure State

- if vault exists but user has no resource-level access, show explanatory access-denied empty state

---

## 6. Admin Resource Detail

### Purpose

Show full resource details for admins and full operators.

### Entry Points

- admin resource list

### Visible Fields

- label
- hostname/IP
- username
- port
- notes
- tags
- credential mode
- reveal/export controls if allowed
- audit summary
- member access summary if permitted

### Hidden or Masked by Role

- workspace-wide security metadata hidden for non-admins

### Actions

- connect
- reveal credential
- copy/export credential
- edit resource
- rotate or replace credential
- open audit trail

### Disabled Actions

- reveal/export disabled when vault policy forbids
- rotate disabled when role lacks admin authority

### Keybindings

- `Enter` connect
- `r` reveal
- `c` copy/export if allowed
- `e` edit
- `A` audit
- `Esc` back to list

### Navigation Destinations

- admin resource list
- edit resource
- audit view

### Empty State

- not applicable

### Error State

- failed to load full details

### Loading State

- loading detail payload

### Permission Failure State

- if privileges changed mid-session, downgrade to restricted detail

---

## 7. Restricted Operator Resource Detail

### Purpose

Show only role-safe details for users who can operate but must not inspect secrets.

### Entry Points

- restricted operator resource list

### Visible Fields

- friendly label
- environment
- service description
- approved connection action
- access request option if elevation exists
- public or low-sensitivity operational notes

### Hidden or Masked by Role

- hostname/IP if policy says hidden
- username
- password/private key material
- export controls
- secret metadata

### Actions

- connect if granted
- request elevated access
- view request status if one exists

### Disabled Actions

- reveal/copy/export hidden
- edit hidden
- connect shown disabled with explanation if approval required

### Keybindings

- `Enter` connect
- `r` request elevation
- `Esc` back to list

### Navigation Destinations

- restricted resource list
- access request flow

### Empty State

- not applicable

### Error State

- failed to load restricted detail

### Loading State

- loading restricted detail payload

### Permission Failure State

- resource exists but user no longer has rights to even view the entry

---

## 8. Invite / Member Management

### Purpose

Let admins invite users, assign vault memberships, and review current members.

### Entry Points

- vault list
- admin resource list
- workspace settings later

### Visible Fields

- current members
- email or identity to invite
- current roles
- invitation status
- last active or device count later

### Hidden or Masked by Role

- hidden entirely for non-admin roles

### Actions

- invite member
- remove member
- resend invite
- change vault role

### Disabled Actions

- owner-only member changes disabled for lower admins if policy requires it

### Keybindings

- `N` invite
- `e` edit role
- `d` remove member
- `Esc` back

### Navigation Destinations

- role assignment flow

### Empty State

- no members beyond owner/admin

### Error State

- failed to load memberships

### Loading State

- loading member list

### Permission Failure State

- not visible to unauthorized roles

---

## 9. Role Assignment

### Purpose

Assign or modify a member's vault role.

### Entry Points

- invite/member management

### Visible Fields

- target user
- current role
- selectable target roles
- plain-language summary of each role

### Hidden or Masked by Role

- hidden for anyone without role-management authority

### Actions

- change role
- cancel change

### Disabled Actions

- roles above current actor's authority disabled with explanation

### Keybindings

- `↑/↓` choose role
- `Enter` apply
- `Esc` cancel

### Navigation Destinations

- back to member management

### Empty State

- not applicable

### Error State

- failed to apply role change

### Loading State

- applying role change

### Permission Failure State

- role no longer manageable by the acting user

---

## 10. Access Request / Approval Flow

### Purpose

Support temporary or approval-based access when the user cannot connect or reveal by default.

### Entry Points

- restricted resource list
- restricted resource detail
- future audit/request inbox

### Visible Fields

- requested resource label
- requested capability such as `connect` or `reveal`
- duration selector
- justification field
- pending/approved/denied status

### Hidden or Masked by Role

- requester should not see hidden metadata they are requesting access to

### Actions

- create request
- cancel pending request
- approve or deny if approver

### Disabled Actions

- request disabled if vault policy forbids self-service requests

### Keybindings

- `Enter` submit
- `a` approve if approver
- `d` deny if approver
- `Esc` cancel or back

### Navigation Destinations

- back to originating resource view
- future `Access Requests` page

### Empty State

- no active requests

### Error State

- failed to submit or process request

### Loading State

- submitting or resolving request

### Permission Failure State

- requester not allowed to request this capability

---

## Cross-Screen Consistency Rules

### Personal vs Teams Switch

The app must make it obvious whether the user is in:

- Personal mode
- Teams mode

### Role-Safe Error Handling

Errors must not leak hidden metadata.
If a restricted operator cannot see hostname/IP, error messages must avoid echoing those fields.

### Role-Safe Search

Search results must respect the same visibility rules as list rows and detail views.

### Connect vs Reveal

Any screen that offers `connect` must independently decide whether `reveal` is:

- visible
- disabled
- hidden

Those decisions must follow the permission matrix, not ad hoc screen logic.
