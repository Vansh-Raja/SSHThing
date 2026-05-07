import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Unlock from './Unlock';

describe('Unlock', () => {
  beforeEach(() => {
    window.sshthing = {
      unlock: vi.fn().mockResolvedValue({ unlocked: true, salt: 'abc', sessionTtlSec: 3600 }),
      listHosts: vi.fn().mockResolvedValue({ hosts: [] }),
      createVault: vi.fn().mockResolvedValue({ ok: true }),
      vaultStatus: vi.fn().mockResolvedValue({ unlocked: false, expiresAt: null }),
      // Satisfy the rest of the SSHThingAPI shape with no-ops so TypeScript is happy
      daemonVersion: vi.fn(),
      changeVaultPassword: vi.fn(),
      lockVault: vi.fn(),
      vacuumVault: vi.fn(),
      getHost: vi.fn(),
      createHost: vi.fn(),
      updateHost: vi.fn(),
      updateHostWithKey: vi.fn(),
      deleteHost: vi.fn(),
      revealCredential: vi.fn(),
      generateKey: vi.fn(),
      importKey: vi.fn(),
      listGroups: vi.fn(),
      createGroup: vi.fn(),
      renameGroup: vi.fn(),
      deleteGroup: vi.fn(),
      openSession: vi.fn(),
      sessionWrite: vi.fn(),
      sessionResize: vi.fn(),
      sessionClose: vi.fn(),
      sessionList: vi.fn(),
      getSettings: vi.fn(),
      setSettings: vi.fn(),
      teamsList: vi.fn(),
      teamsHostsList: vi.fn(),
      teamsHostsCreate: vi.fn(),
      teamsHostsUpdate: vi.fn(),
      teamsHostsDelete: vi.fn(),
      teamsMembersList: vi.fn(),
      teamsMembersInvite: vi.fn(),
      teamsMembersUpdateRole: vi.fn(),
      teamsMembersRemove: vi.fn(),
      teamsInvitesList: vi.fn(),
      teamsInvitesAccept: vi.fn(),
      teamsInvitesRevoke: vi.fn(),
      teamsAuditList: vi.fn(),
      teamsCreate: vi.fn(),
      teamsRename: vi.fn(),
      teamsDelete: vi.fn(),
      teamsReorder: vi.fn(),
      teamsLeave: vi.fn(),
      teamsHostsRevealShared: vi.fn(),
      teamsHostsRosterList: vi.fn(),
      teamsHostsRevealMember: vi.fn(),
      teamsHostsDeleteMemberCredential: vi.fn(),
      teamsHostsUpsertMyCredential: vi.fn(),
      teamsHostsImportPersonalPreview: vi.fn(),
      teamsHostsImportPersonalCommit: vi.fn(),
      tokensList: vi.fn(),
      tokensCreate: vi.fn(),
      tokensRevoke: vi.fn(),
      tokensDeleteRevoked: vi.fn(),
      healthProbe: vi.fn(),
      healthList: vi.fn(),
      mountStart: vi.fn(),
      mountStop: vi.fn(),
      mountList: vi.fn(),
      mountCheckPrereqs: vi.fn(),
      transferUpload: vi.fn(),
      transferDownload: vi.fn(),
      transferCancel: vi.fn(),
      sessionExec: vi.fn(),
      authStartSignIn: vi.fn(),
      authOpenBrowser: vi.fn(),
      authPollSignIn: vi.fn(),
      authSignOut: vi.fn(),
      authSession: vi.fn(),
      authTokenForRenderer: vi.fn(),
      syncStatus: vi.fn(),
      syncNow: vi.fn(),
      syncConfigure: vi.fn(),
      syncEvents: vi.fn(),
      syncDevices: vi.fn(),
      syncForgetDevice: vi.fn(),
      syncTestGit: vi.fn(),
      keyringHealthCheck: vi.fn(),
      installUpdate: vi.fn(),
      checkForUpdates: vi.fn(),
      openPath: vi.fn(),
      chooseDirectory: vi.fn(),
      onNotification: vi.fn(() => () => {}),
      onMenuCommand: vi.fn(() => () => {}),
      onDaemonExited: vi.fn(() => () => {}),
      onUpdateAvailable: vi.fn(() => () => {}),
      onUpdateDownloaded: vi.fn(() => () => {}),
    } as unknown as Window['sshthing'];
  });

  it('renders the password input', () => {
    render(
      <MemoryRouter>
        <Unlock />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText(/vault password/i)).toBeInTheDocument();
  });

  it('calls window.sshthing.unlock when submitting the form', async () => {
    render(
      <MemoryRouter>
        <Unlock />
      </MemoryRouter>,
    );

    const passwordInput = screen.getByLabelText(/vault password/i);
    const submitButton = screen.getByRole('button', { name: /unlock/i });

    fireEvent.change(passwordInput, { target: { value: 'secret123' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(window.sshthing.unlock).toHaveBeenCalledWith('secret123');
    });
  });
});
