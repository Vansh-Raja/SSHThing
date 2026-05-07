/**
 * Electron main process entry point.
 *
 * Responsibilities:
 * 1. Spawn the sshthing-daemon binary.
 * 2. Read daemon.token once the daemon has started.
 * 3. Open the BrowserWindow and load the renderer.
 * 4. Expose the DaemonClient to the preload via IPC.
 * 5. Kill the daemon when all windows are closed.
 */
import { app, BrowserWindow, dialog, ipcMain, Menu, shell, type MenuItemConstructorOptions } from 'electron';
import { autoUpdater, type UpdateInfo } from 'electron-updater';
import * as path from 'path';
import * as fs from 'fs';
import * as child_process from 'child_process';
import windowStateKeeper from 'electron-window-state';
import { DaemonClient, defaultSocketPath, defaultTokenPath, type HostCreate, type HostUpdate, type AppSettings, type CreateTeamHostRequest, type UpdateTeamHostRequest, type TeamRole, type TokenHostGrant, type TransferParams, type SyncConfigureParams, type UpsertMyCredentialRequest, type ImportPersonalHostCommitRequest } from './daemon';

let daemonProc: child_process.ChildProcess | null = null;
let client: DaemonClient | null = null;

function resolveDaemonBinary(): string {
  // In production: packaged binary lives next to resources.
  // In development: look in ../bin/sshthing-daemon relative to this file,
  // then fall back to a PATH lookup.
  const devBin = path.resolve(__dirname, '..', '..', 'bin', 'sshthing-daemon');
  if (fs.existsSync(devBin)) {
    return devBin;
  }
  // Also check the project root for a freshly-built binary.
  const rootBin = path.resolve(__dirname, '..', '..', '..', '..', 'sshthing-daemon');
  if (fs.existsSync(rootBin)) {
    return rootBin;
  }
  // Last resort: assume it's on PATH.
  return 'sshthing-daemon';
}

async function readTokenWithRetry(tokenPath: string, maxRetries = 80, intervalMs = 100): Promise<string> {
  // The daemon binary can take a couple of seconds to start, especially the
  // first run after rebuild on macOS Gatekeeper. We retry generously (8s
  // total) before giving up.
  for (let i = 0; i < maxRetries; i++) {
    try {
      const stat = fs.statSync(tokenPath);
      // Only trust files newer than ~10s ago — avoids reading a stale token
      // from a previous run that the current daemon hasn't overwritten yet.
      const ageMs = Date.now() - stat.mtimeMs;
      if (ageMs < 10_000) {
        const token = fs.readFileSync(tokenPath, 'utf8').trim();
        if (token.length === 64) return token; // 32-byte hex = 64 chars
      }
    } catch {
      // file not written yet
    }
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  throw new Error(`daemon token not available after ${maxRetries * intervalMs}ms`);
}

async function startDaemon(): Promise<DaemonClient> {
  const bin = resolveDaemonBinary();
  console.log('[main] spawning daemon:', bin);

  // Remove any stale token/socket from a previous run so we don't race-read
  // the old token before the daemon writes a fresh one.
  const stalePaths = [defaultTokenPath(), defaultSocketPath()];
  for (const p of stalePaths) {
    try {
      fs.unlinkSync(p);
    } catch {
      // not present — fine
    }
  }

  daemonProc = child_process.spawn(bin, [], {
    stdio: ['ignore', 'pipe', 'pipe'],
    detached: false,
  });

  daemonProc.stdout?.on('data', (d: Buffer) => {
    process.stdout.write('[daemon] ' + d.toString());
  });
  daemonProc.stderr?.on('data', (d: Buffer) => {
    process.stderr.write('[daemon] ' + d.toString());
  });
  daemonProc.on('exit', (code, signal) => {
    console.error(`[main] daemon exited: code=${code} signal=${signal}`);
    daemonProc = null;
    // Notify all open renderer windows so they can surface a reconnect banner.
    BrowserWindow.getAllWindows().forEach((w) => {
      if (!w.isDestroyed()) {
        w.webContents.send('app:daemon-exited');
      }
    });
  });

  // Wait for token file to appear.
  const tokenPath = defaultTokenPath();
  const token = await readTokenWithRetry(tokenPath);
  console.log('[main] daemon token read');

  const sockPath = defaultSocketPath();
  const c = new DaemonClient();
  // Retry connect — daemon might not have bound the socket yet (5s budget).
  for (let i = 0; i < 50; i++) {
    try {
      await c.connect(sockPath, token);
      console.log('[main] connected to daemon socket');
      return c;
    } catch {
      await new Promise((r) => setTimeout(r, 100));
    }
  }
  throw new Error('failed to connect to daemon socket after retries');
}

function killDaemon(): void {
  if (daemonProc) {
    console.log('[main] killing daemon');
    daemonProc.kill('SIGTERM');
    daemonProc = null;
  }
  client?.disconnect();
  client = null;
}

function createWindow(): BrowserWindow {
  const state = windowStateKeeper({
    defaultWidth: 1280,
    defaultHeight: 820,
  });

  const win = new BrowserWindow({
    width: state.width,
    height: state.height,
    x: state.x,
    y: state.y,
    minWidth: 880,
    minHeight: 540,
    webPreferences: {
      preload: path.join(__dirname, '..', 'preload', 'index.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
    title: 'SSHThing',
    backgroundColor: '#0e1117',
    // macOS uses the app bundle icon; Linux/Windows need an explicit path.
    icon: process.platform === 'darwin' ? undefined : path.resolve(__dirname, '..', '..', 'assets', 'icon.png'),
    // On macOS, integrate the traffic lights into our custom topbar so
    // we don't end up with a redundant OS title row above our chrome.
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    trafficLightPosition: { x: 14, y: 14 },
  });

  state.manage(win);

  win.loadFile(path.join(__dirname, '..', 'renderer', 'index.html'));
  // In dev, open DevTools.
  // win.webContents.openDevTools();

  return win;
}

// IPC handlers — called by the preload via ipcMain.invoke.
function registerIPC(c: DaemonClient): void {
  ipcMain.handle('daemon:version', () => c.daemonVersion());
  ipcMain.handle('vault:unlock', (_e, password: string) => c.unlock(password));
  ipcMain.handle('vault:status', () => c.vaultStatus());
  ipcMain.handle('vault:create', (_e, password: string) => c.createVault(password));
  ipcMain.handle('vault:changePassword', (_e, oldPassword: string, newPassword: string) =>
    c.changeVaultPassword(oldPassword, newPassword)
  );
  ipcMain.handle('vault:lock', () => c.lockVault());
  ipcMain.handle('vault:vacuum', () => c.vacuumVault());
  ipcMain.handle('hosts:list', (_e, query?: string) => c.listHosts(query));
  ipcMain.handle('hosts:get', (_e, id: string) => c.getHost(id));
  ipcMain.handle('hosts:create', (_e, host: HostCreate) => c.createHost(host));
  ipcMain.handle('hosts:update', (_e, host: HostUpdate) => c.updateHost(host));
  ipcMain.handle('hosts:updateWithKey', (_e, host: HostUpdate & { plainKey: string }) => c.updateHostWithKey(host));
  ipcMain.handle('hosts:delete', (_e, id: string) => c.deleteHost(id));
  ipcMain.handle('hosts:revealCredential', (_e, hostId: string) => c.revealCredential(hostId));
  ipcMain.handle('hosts:generateKey', (_e, keyType: string, comment: string) =>
    c.generateKey(keyType, comment)
  );
  ipcMain.handle('hosts:importKey', (_e, format: string, blob: string, label: string, hostname: string, username: string, port: number) =>
    c.importKey(format, blob, label, hostname, username, port)
  );
  ipcMain.handle('groups:list', () => c.listGroups());
  ipcMain.handle('groups:create', (_e, name: string) => c.createGroup(name));
  ipcMain.handle('groups:rename', (_e, oldName: string, newName: string) => c.renameGroup(oldName, newName));
  ipcMain.handle('groups:delete', (_e, name: string) => c.deleteGroup(name));
  ipcMain.handle('session:open', (_e, hostId: string, cols: number, rows: number, term?: string) =>
    c.openSession(hostId, cols, rows, term)
  );
  ipcMain.handle('session:write', (_e, sessionId: string, data: number[]) =>
    c.sessionWrite(sessionId, new Uint8Array(data))
  );
  ipcMain.handle('session:resize', (_e, sessionId: string, cols: number, rows: number) =>
    c.sessionResize(sessionId, cols, rows)
  );
  ipcMain.handle('session:close', (_e, sessionId: string) => c.sessionClose(sessionId));
  ipcMain.handle('session:list', () => c.sessionList());
  ipcMain.handle('settings:get', () => c.getSettings());
  ipcMain.handle('settings:set', (_e, patch: Partial<AppSettings>) => c.setSettings(patch));

  // ---- Teams IPC ----
  ipcMain.handle('teams:list', () => c.teamsList());
  ipcMain.handle('teams:hosts:list', (_e, teamId: string) => c.teamsHostsList(teamId));
  ipcMain.handle('teams:hosts:create', (_e, teamId: string, req: CreateTeamHostRequest) => c.teamsHostsCreate(teamId, req));
  ipcMain.handle('teams:hosts:update', (_e, hostId: string, req: UpdateTeamHostRequest) => c.teamsHostsUpdate(hostId, req));
  ipcMain.handle('teams:hosts:delete', (_e, hostId: string) => c.teamsHostsDelete(hostId));
  ipcMain.handle('teams:members:list', (_e, teamId: string) => c.teamsMembersList(teamId));
  ipcMain.handle('teams:members:invite', (_e, teamId: string, email: string, role: TeamRole) => c.teamsMembersInvite(teamId, email, role));
  ipcMain.handle('teams:members:updateRole', (_e, teamId: string, memberId: string, role: TeamRole) => c.teamsMembersUpdateRole(teamId, memberId, role));
  ipcMain.handle('teams:members:remove', (_e, teamId: string, memberId: string) => c.teamsMembersRemove(teamId, memberId));
  ipcMain.handle('teams:invites:list', (_e, teamId: string) => c.teamsInvitesList(teamId));
  ipcMain.handle('teams:invites:accept', (_e, inviteId: string) => c.teamsInvitesAccept(inviteId));
  ipcMain.handle('teams:invites:revoke', (_e, teamId: string, inviteId: string) => c.teamsInvitesRevoke(teamId, inviteId));
  ipcMain.handle('teams:audit:list', (_e, teamId: string) => c.teamsAuditList(teamId));
  ipcMain.handle('teams:create', (_e, name: string) => c.teamsCreate(name));
  ipcMain.handle('teams:rename', (_e, teamId: string, name: string) => c.teamsRename(teamId, name));
  ipcMain.handle('teams:delete', (_e, teamId: string) => c.teamsDelete(teamId));
  ipcMain.handle('teams:reorder', (_e, teamIds: string[]) => c.teamsReorder(teamIds));
  ipcMain.handle('teams:leave', (_e, teamId: string) => c.teamsLeave(teamId));

  // ---- Team credentials IPC ----
  ipcMain.handle('teams:hosts:credentials:revealShared', (_e, hostId: string) => c.teamsHostsRevealShared(hostId));
  ipcMain.handle('teams:hosts:credentials:rosterList', (_e, hostId: string) => c.teamsHostsRosterList(hostId));
  ipcMain.handle('teams:hosts:credentials:revealMember', (_e, hostId: string, memberId: string) => c.teamsHostsRevealMember(hostId, memberId));
  ipcMain.handle('teams:hosts:credentials:deleteMember', (_e, hostId: string, memberId: string) => c.teamsHostsDeleteMemberCredential(hostId, memberId));
  ipcMain.handle('teams:hosts:credentials:upsertMine', (_e, hostId: string, req: UpsertMyCredentialRequest) => c.teamsHostsUpsertMyCredential(hostId, req));
  ipcMain.handle('teams:hosts:importPersonal:preview', (_e, personalHostId: string, teamId: string) => c.teamsHostsImportPersonalPreview(personalHostId, teamId));
  ipcMain.handle('teams:hosts:importPersonal:commit', (_e, req: ImportPersonalHostCommitRequest) => c.teamsHostsImportPersonalCommit(req));

  // ---- Tokens IPC ----
  ipcMain.handle('tokens:list', () => c.tokensList());
  ipcMain.handle('tokens:create', (_e, name: string, grants: TokenHostGrant[]) => c.tokensCreate(name, grants));
  ipcMain.handle('tokens:revoke', (_e, tokenId: string) => c.tokensRevoke(tokenId));
  ipcMain.handle('tokens:deleteRevoked', (_e, tokenId: string) => c.tokensDeleteRevoked(tokenId));

  // ---- Health IPC ----
  ipcMain.handle('health:probe', (_e, hostId: string) => c.healthProbe(hostId));
  ipcMain.handle('health:list', () => c.healthList());

  // ---- Mount IPC ----
  ipcMain.handle('mount:start', (_e, hostId: string, remotePath: string) => c.mountStart(hostId, remotePath));
  ipcMain.handle('mount:stop', (_e, hostId: string) => c.mountStop(hostId));
  ipcMain.handle('mount:list', () => c.mountList());
  ipcMain.handle('mount:checkPrereqs', () => c.mountCheckPrereqs());

  // ---- Transfer IPC ----
  ipcMain.handle('transfer:upload', (_e, params: TransferParams) => c.transferUpload(params));
  ipcMain.handle('transfer:download', (_e, params: TransferParams) => c.transferDownload(params));
  ipcMain.handle('transfer:cancel', (_e, transferId: string) => c.transferCancel(transferId));

  // ---- Exec IPC ----
  ipcMain.handle('session:exec', (_e, hostId: string, cmd: string, timeoutMs?: number) =>
    c.sessionExec(hostId, cmd, timeoutMs)
  );

  // ---- Auth IPC ----
  ipcMain.handle('auth:startSignIn', () => c.authStartSignIn());
  ipcMain.handle('auth:openBrowser', (_e, url: string) => c.authOpenBrowser(url));
  ipcMain.handle('auth:pollSignIn', (_e, sessionId: string, pollSecret: string) =>
    c.authPollSignIn(sessionId, pollSecret)
  );
  ipcMain.handle('auth:signOut', () => c.authSignOut());
  ipcMain.handle('auth:session', () => c.authSession());
  ipcMain.handle('auth:tokenForRenderer', () => c.authTokenForRenderer());

  // ---- Sync IPC ----
  ipcMain.handle('sync:status', () => c.syncStatus());
  ipcMain.handle('sync:now', () => c.syncNow());
  ipcMain.handle('sync:configure', (_e, params: SyncConfigureParams) => c.syncConfigure(params));
  ipcMain.handle('sync:events', () => c.syncEvents());
  ipcMain.handle('sync:devices', () => c.syncDevices());
  ipcMain.handle('sync:forgetDevice', (_e, deviceId: string) => c.syncForgetDevice(deviceId));
  ipcMain.handle('sync:testGit', (_e, repoUrl: string, sshKeyPath: string) => c.syncTestGit(repoUrl, sshKeyPath));
  ipcMain.handle('keyring:healthCheck', () => c.keyringHealthCheck());

  // ---- System IPC ----
  // shell.openPath returns a string: empty on success, error message on failure.
  ipcMain.handle('system:openPath', (_e, filePath: string) => shell.openPath(filePath));

  // ---- Dialog IPC ----
  ipcMain.handle('dialog:open-directory', async () => {
    const result = await dialog.showOpenDialog({
      properties: ['openDirectory', 'createDirectory'],
    });
    return { canceled: result.canceled, path: result.filePaths[0] ?? null };
  });

  // ---- Update IPC ----
  ipcMain.handle('update:install', () => {
    autoUpdater.quitAndInstall();
  });

  ipcMain.handle('update:check', () => {
    void autoUpdater.checkForUpdatesAndNotify();
  });

  // Forward daemon notifications to renderer.
  c.on('notification', (method: string, params: unknown) => {
    BrowserWindow.getAllWindows().forEach((w) => {
      w.webContents.send('daemon:notification', method, params);
    });
  });
}

// ──────────────────────────────────────────────────────────
// App menu
// ──────────────────────────────────────────────────────────

/**
 * Send a menu command to the focused renderer window. The renderer listens
 * via `window.sshthing.onMenuCommand`.
 */
function sendMenuCommand(cmd: string): void {
  const win = BrowserWindow.getFocusedWindow() ?? BrowserWindow.getAllWindows()[0];
  win?.webContents.send('app:menu-command', cmd);
}

function buildAppMenu(): void {
  const isMac = process.platform === 'darwin';

  const macAppMenu: MenuItemConstructorOptions = {
    label: 'SSHThing',
    submenu: [
      {
        label: 'About SSHThing',
        click: () => sendMenuCommand('open-about'),
      },
      { type: 'separator' },
      {
        label: 'Settings',
        accelerator: 'CmdOrCtrl+,',
        click: () => sendMenuCommand('open-settings'),
      },
      {
        label: 'Lock vault',
        accelerator: 'CmdOrCtrl+L',
        click: () => sendMenuCommand('lock-vault'),
      },
      {
        label: 'Account',
        click: () => sendMenuCommand('open-account'),
      },
      {
        label: 'Sign out',
        click: () => sendMenuCommand('sign-out'),
      },
      { type: 'separator' },
      { role: 'hide' },
      { role: 'hideOthers' },
      { role: 'unhide' },
      { type: 'separator' },
      { role: 'quit' },
    ],
  };

  const fileMenu: MenuItemConstructorOptions = {
    label: 'File',
    submenu: [
      {
        label: 'New tab',
        click: () => sendMenuCommand('new-tab'),
      },
      ...(isMac
        ? []
        : [
            { type: 'separator' as const },
            {
              label: 'Settings',
              accelerator: 'CmdOrCtrl+,',
              click: () => sendMenuCommand('open-settings'),
            },
            {
              label: 'Lock vault',
              accelerator: 'CmdOrCtrl+L',
              click: () => sendMenuCommand('lock-vault'),
            },
            {
              label: 'Sign out',
              click: () => sendMenuCommand('sign-out'),
            },
            { type: 'separator' as const },
            { role: 'quit' as const },
          ]),
    ],
  };

  const editMenu: MenuItemConstructorOptions = {
    label: 'Edit',
    submenu: [
      { role: 'undo' },
      { role: 'redo' },
      { type: 'separator' },
      { role: 'cut' },
      { role: 'copy' },
      { role: 'paste' },
      { role: 'selectAll' },
    ],
  };

  const viewMenu: MenuItemConstructorOptions = {
    label: 'View',
    submenu: [
      { role: 'reload' },
      { role: 'forceReload' },
      { role: 'toggleDevTools' },
      { type: 'separator' },
      { role: 'zoomIn' },
      { role: 'zoomOut' },
      { role: 'resetZoom' },
      { type: 'separator' },
      { role: 'togglefullscreen' },
    ],
  };

  const windowMenu: MenuItemConstructorOptions = {
    label: 'Window',
    submenu: [
      { role: 'minimize' },
      { role: 'zoom' },
      ...(isMac
        ? [{ type: 'separator' as const }, { role: 'front' as const }]
        : [{ role: 'close' as const }]),
    ],
  };

  const helpMenu: MenuItemConstructorOptions = {
    label: 'Help',
    submenu: [
      {
        label: 'Open help',
        accelerator: isMac ? 'Cmd+?' : 'F1',
        click: () => sendMenuCommand('open-help'),
      },
    ],
  };

  const template: MenuItemConstructorOptions[] = [
    ...(isMac ? [macAppMenu] : []),
    fileMenu,
    editMenu,
    viewMenu,
    windowMenu,
    helpMenu,
  ];

  Menu.setApplicationMenu(Menu.buildFromTemplate(template));
}

app.whenReady().then(async () => {
  try {
    client = await startDaemon();
    registerIPC(client);
  } catch (err) {
    console.error('[main] failed to start daemon:', err);
    // Continue anyway so the user can see the error in the window.
  }

  buildAppMenu();
  const win = createWindow();

  // Auto-updater wiring
  autoUpdater.on('update-available', (info: UpdateInfo) => {
    BrowserWindow.getAllWindows().forEach((w) => {
      if (!w.isDestroyed()) {
        w.webContents.send('app:update-available', info);
      }
    });
  });

  autoUpdater.on('update-downloaded', (info: UpdateInfo) => {
    BrowserWindow.getAllWindows().forEach((w) => {
      if (!w.isDestroyed()) {
        w.webContents.send('app:update-downloaded', info);
      }
    });
  });

  // Check for updates on start (don't block window creation).
  setTimeout(() => {
    void autoUpdater.checkForUpdatesAndNotify();
  }, 3000);

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  // On macOS apps usually stay alive until Cmd+Q, so we only kill the
  // daemon on before-quit. On other platforms, closing the window quits
  // the app, which triggers before-quit and tears the daemon down there.
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

let isQuitting = false;

app.on('before-quit', (event) => {
  if (isQuitting) {
    killDaemon();
    return;
  }

  // We need to do async work (check mounts, maybe show a dialog), so prevent
  // default now and call app.quit() manually once the decision is made.
  event.preventDefault();

  (async () => {
    try {
      if (!client) {
        isQuitting = true;
        app.quit();
        return;
      }

      const { mounts } = await client.mountList();
      if (mounts.length === 0) {
        isQuitting = true;
        app.quit();
        return;
      }

      const { response } = await dialog.showMessageBox({
        type: 'warning',
        title: 'Active mounts',
        message: 'You have active SSHFS mounts. Quitting will unmount them.',
        buttons: ['Unmount & Quit', 'Leave Mounted', 'Cancel'],
        defaultId: 0,
        cancelId: 2,
      });

      if (response === 2) {
        // Cancel — keep the app running.
        return;
      }

      if (response === 0) {
        // Unmount & Quit
        for (const m of mounts) {
          try {
            await client.mountStop(m.hostId);
          } catch {
            // ignore individual unmount failures
          }
        }
      }

      // response === 1: Leave Mounted — just kill daemon and quit.
      isQuitting = true;
      app.quit();
    } catch {
      isQuitting = true;
      app.quit();
    }
  })();
});
