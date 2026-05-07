import { useNavigate } from 'react-router-dom';
import { useIncomingInvites } from '../hooks/useIncomingInvites';
import { BellIcon } from './icons';

export default function InvitesBadge() {
  const { count } = useIncomingInvites();
  const navigate = useNavigate();

  return (
    <button
      type="button"
      className="topbar__invites"
      title={count > 0 ? `${count} pending invite${count === 1 ? '' : 's'}` : 'Invites'}
      onClick={() => navigate('/teams')}
    >
      <BellIcon />
      {count > 0 && <span className="topbar__invites-badge" />}
    </button>
  );
}
