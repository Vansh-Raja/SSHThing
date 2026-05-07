/**
 * useTeamRole — derives the current viewer's role in a given team.
 *
 * Returns the role from the TeamSummary.role field (populated when fetching
 * the teams list). Falls back to 'member' if unknown.
 *
 * Wave 2B note: components that need fine-grained permission checks (e.g.
 * canManageHosts on TeamHostsTab, canRevealSecrets on credential rows) should
 * use the per-host fields (TeamHost.canManageHosts, etc.) rather than this
 * hook. This hook is for UI-level gating of team-level settings actions.
 */
export function useTeamRole(team: TeamSummary | null): TeamRole {
  return team?.role ?? 'member';
}
