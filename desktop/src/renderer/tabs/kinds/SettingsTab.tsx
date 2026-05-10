/**
 * SettingsTab — wraps the existing Settings page so it renders inside
 * the workspace tab system instead of as a routed page. This is a thin
 * shim today; the underlying Settings component is reused untouched so
 * the migration doesn't have to rewrite that screen.
 */
import Settings from '../../pages/Settings';
import type { TabContentProps } from '../registry';

export default function SettingsTab(_props: TabContentProps) {
  return <Settings />;
}
