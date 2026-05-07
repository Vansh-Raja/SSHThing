/**
 * TeamSwitcher — dropdown for selecting the active team, shown in the top bar
 * when the user is signed in and has at least one team.
 */
import DropdownMenu, { type MenuItemDef } from '../../ui/DropdownMenu';

type TeamSwitcherProps = {
  teams: TeamSummary[];
  activeTeamId: string | null;
  onSelect: (team: TeamSummary) => void;
};

export default function TeamSwitcher({ teams, activeTeamId, onSelect }: TeamSwitcherProps) {
  const active = teams.find((t) => t.id === activeTeamId);
  const label = active?.name ?? 'Select team';

  const items: MenuItemDef[] = teams.map((t) => ({
    kind: 'item' as const,
    label: t.name,
    onClick: () => onSelect(t),
    disabled: t.id === activeTeamId,
  }));

  if (teams.length === 0) return null;

  return (
    <DropdownMenu
      trigger={
        <button
          type="button"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            background: 'var(--paper-2)',
            border: '1px solid var(--line)',
            borderRadius: 4,
            padding: '4px 10px',
            fontSize: 12,
            fontWeight: 600,
            color: 'var(--ink)',
            cursor: 'pointer',
            whiteSpace: 'nowrap',
            maxWidth: 180,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {label}
          <span style={{ color: 'var(--muted)', fontSize: 10 }}>▾</span>
        </button>
      }
      items={items}
    />
  );
}
