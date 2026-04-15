# SSHThing Teams: MVP Scope

## Summary

This document defines what belongs in the first shippable version of SSHThing Teams and what is intentionally deferred.

The MVP goal is to prove:

- account-backed Teams mode works
- workspaces and vaults are usable in the TUI
- role-based visibility and access are coherent
- restricted operators can connect without being treated like admins

## MVP In Scope

### Identity and Entry

- Teams mode sign-in entry from the TUI
- account-backed access to Teams mode
- workspace list and workspace switching

### Workspace and Vault Structure

- workspace creation
- vault listing
- vault selection

### Membership and Roles

- invites
- member list
- role assignment
- member removal

### Resource Management

- create shared resource entries
- list shared resources in a vault
- role-specific resource rendering
- admin and restricted-operator detail views

### Access Behavior

- connect vs reveal separation
- restricted operator role
- personal-credential and shared-credential conceptual support in product design

### TUI

- Teams entry flow
- workspace switcher
- vault list
- resource list for admin
- resource list for restricted operator
- resource detail for admin
- resource detail for restricted operator
- invite/member flow
- role assignment flow

## MVP Out of Scope

### Access Governance

- temporary elevated access grants
- full access request approvals system
- approval inbox

### Advanced Security

- brokered sessions
- ephemeral SSH certificates
- session recording
- moderated sessions
- device approval workflows

### Advanced Integrations

- GitHub org/repo bootstrap
- GH CLI-assisted setup
- Git-backed team backend
- enterprise SSO/SCIM

### Self-Hosted Productization

- self-hosted Teams deployment UX
- admin install flow for self-hosted deployments

## Post-MVP Scope

### Post-MVP 1

- access requests
- temporary grants
- audit feed
- request and approval inbox

### Post-MVP 2

- GitHub-assisted setup
- optional Git-backed storage mode
- self-hosted Teams mode

### Long-Term

- brokered access
- ephemeral credentials
- stronger no-export guarantees
- enterprise identity integrations

## Brokered Access Position

Brokered access is explicitly **not** an MVP requirement.

It is a strategic direction because it best supports:

- connect without reveal
- stronger offboarding
- better auditability

But it should not block the first Teams release.

## MVP Product Guarantees

The MVP should still guarantee:

- Personal mode remains intact
- Teams mode is clearly separate
- role-based visibility is enforced in the UI
- restricted users are not shown raw credential actions
- admin/operator/restricted-operator experiences are visibly different

## MVP Review Checklist

Before declaring the MVP scope frozen, confirm:

- the 10 mock screens are complete
- the permission matrix is frozen
- the resource model is frozen
- the wording around shared credentials vs restricted access is honest
- no MVP feature assumes brokered sessions exist yet
