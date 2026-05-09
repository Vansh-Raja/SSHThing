/**
 * HelpOverlay — keyboard shortcut cheatsheet.
 * Opens via `?` shortcut or `/help` command.
 * Data-driven: add to SHORTCUT_GROUPS to extend.
 */
import Modal from '../ui/Modal';

interface Shortcut {
  keys: string[];
  desc: string;
}

interface ShortcutGroup {
  area: string;
  shortcuts: Shortcut[];
}

const SHORTCUT_GROUPS: ShortcutGroup[] = [
  {
    area: 'Global',
    shortcuts: [
      { keys: ['⌘K'], desc: 'Open command palette' },
      { keys: [':'], desc: 'Open command palette (command mode)' },
      { keys: ['/'], desc: 'Open spotlight (arming chords)' },
      { keys: ['Esc'], desc: 'Close overlay / cancel' },
      { keys: ['⌘,'], desc: 'Open settings' },
      { keys: ['⌘L'], desc: 'Lock vault' },
      { keys: ['⌘⇧T'], desc: 'Switch team' },
      { keys: ['?'], desc: 'Open this help overlay' },
    ],
  },
  {
    area: 'Hosts',
    shortcuts: [
      { keys: ['⌘T'], desc: 'Open new terminal tab' },
      { keys: ['⌘W'], desc: 'Close current tab' },
      { keys: ['⌘1', '…', '⌘9'], desc: 'Switch to tab N' },
      { keys: ['⌘R'], desc: 'Refresh / reconnect tab' },
    ],
  },
  {
    area: 'Drawer / Form',
    shortcuts: [
      { keys: ['V'], desc: 'Open key editor (paste mode)' },
      { keys: ['Esc'], desc: 'Cancel / close drawer' },
      { keys: ['⌘S'], desc: 'Save form' },
      { keys: ['Tab'], desc: 'Focus next field' },
      { keys: ['⇧Tab'], desc: 'Focus previous field' },
    ],
  },
  {
    area: 'Palette',
    shortcuts: [
      { keys: ['↑', '↓'], desc: 'Navigate results' },
      { keys: ['↵'], desc: 'Run / connect' },
      { keys: ['/'], desc: 'Enter slash-command mode' },
      { keys: ['/q'], desc: 'Quit app' },
      { keys: ['/n'], desc: 'New host' },
      { keys: ['/e'], desc: 'Edit selected host' },
      { keys: ['/d'], desc: 'Delete selected host' },
      { keys: ['/s'], desc: 'Cycle sort order' },
    ],
  },
  {
    area: 'Spotlight',
    shortcuts: [
      { keys: ['S'], desc: 'Search hosts' },
      { keys: ['N'], desc: 'New host' },
      { keys: ['C'], desc: 'Connect' },
      { keys: ['M'], desc: 'Mount' },
      { keys: ['E'], desc: 'Execute' },
      { keys: ['H'], desc: 'Health check' },
      { keys: ['T'], desc: 'Tokens' },
      { keys: ['?'], desc: 'Help' },
    ],
  },
  {
    area: 'Settings',
    shortcuts: [
      { keys: ['↑', '↓'], desc: 'Navigate categories' },
      { keys: ['V'], desc: 'Vault' },
      { keys: ['A'], desc: 'Appearance' },
      { keys: ['S'], desc: 'SSH Defaults' },
      { keys: ['Y'], desc: 'Sync' },
      { keys: ['U'], desc: 'Updates' },
      { keys: ['H'], desc: 'Health' },
    ],
  },
];

type HelpOverlayProps = {
  open: boolean;
  onClose: () => void;
};

export default function HelpOverlay({ open, onClose }: HelpOverlayProps) {
  return (
    <Modal open={open} onClose={onClose} title="Keyboard shortcuts" maxWidth={520}>
      <div className="help-overlay">
        {SHORTCUT_GROUPS.map((group) => (
          <section key={group.area} className="help-group">
            <h3 className="help-group__area">{group.area}</h3>
            <ul className="help-group__list">
              {group.shortcuts.map((sc) => (
                <li key={sc.desc} className="help-shortcut">
                  <span className="help-shortcut__keys">
                    {sc.keys.map((k, i) => (
                      <span key={i}>
                        {i > 0 && <span className="help-shortcut__sep">/</span>}
                        <kbd className="help-kbd">{k}</kbd>
                      </span>
                    ))}
                  </span>
                  <span className="help-shortcut__desc">{sc.desc}</span>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </Modal>
  );
}
