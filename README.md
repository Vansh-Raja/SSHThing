# SSH Manager TUI

A modern, secure Terminal User Interface (TUI) for managing SSH connections built with Go and Bubble Tea.

![Version](https://img.shields.io/badge/version-0.1.0--mvp-blue)
![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

### ✨ Current (MVP)
- 🎨 **Beautiful TUI** - Modern, responsive interface built with Bubble Tea
- ⌨️ **Intuitive Navigation** - Arrow keys, keyboard shortcuts, and search
- 🏠 **Host Management** - Add, edit, and delete SSH host configurations
- 🔑 **Dual Key Support** - Generate new SSH keys OR paste existing ones
- 🔍 **Real-time Search** - Filter hosts by hostname, username, or notes
- 📋 **Two-panel Layout** - Host list + detailed view
- ⚡ **Fast & Lightweight** - Single 4MB binary, no dependencies

### 🚀 Planned (Phase 2+)
- 🔐 **Dual-layer Encryption** - SQLCipher (AES-256) + Fernet per-key encryption
- 💾 **Persistent Storage** - Encrypted SQLite database
- 🔌 **SSH Connection** - Direct SSH with auto-decrypted keys
- 🔒 **Secure Key Handling** - Keys decrypted only in memory, temp files auto-deleted
- 📊 **Connection History** - Track last connected timestamps
- 🎯 **Vim Keybindings** - Optional Vim-style navigation (j/k/gg/G)

## Screenshots

```
┌──────────────────────────────────────────────┐
│ 🔐 SSH Manager                   [?] Help    │
├────────────────┬─────────────────────────────┤
│                │                             │
│  HOSTS         │   DETAILS                   │
│  ──────────    │   ─────────────────────    │
│ ⚡ web-prod-1  │   Hostname: web-prod-1    │
│   db-server    │   Username: ec2-user      │
│   staging      │   Port: 22                │
│   backup-nas   │   Status: Ready ✓         │
│                │   Key Type: ed25519       │
│  Filter: ___   │   Last SSH: 1 day ago     │
│                │                             │
├────────────────┴─────────────────────────────┤
│ [↑/↓] Navigate [Enter] Connect [a] Add [q] Quit │
└──────────────────────────────────────────────┘
```

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/Vansh-Raja/SSHThing.git
cd SSHThing

# Build the binary
go build -o ssh-manager

# Run
./ssh-manager
```

### Requirements
- Go 1.21 or higher
- Terminal with 256 color support (most modern terminals)

## Usage

### Keyboard Shortcuts

#### Main View
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate host list |
| `Ctrl+U/D` | Page up/down |
| `Home/End` or `g/G` | Jump to top/bottom |
| `Enter` | Connect to selected host (coming soon) |
| `a` or `Ctrl+N` | Add new host |
| `e` | Edit selected host |
| `d` or `Delete` | Delete selected host |
| `/` or `Ctrl+F` | Search/filter hosts |
| `?` | Show help |
| `q` or `Ctrl+C` | Quit |

#### Modal Forms
| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Enter` | Submit form |
| `Esc` | Cancel |
| `Space` | Toggle options |

### Adding a Host

1. Press `a` to open the "Add Host" modal
2. Fill in the details:
   - **Hostname**: SSH server address (e.g., `example.com` or `192.168.1.100`)
   - **Username**: SSH username (e.g., `ubuntu`, `ec2-user`)
   - **Port**: SSH port (default: `22`)
   - **Notes**: Optional notes about the host
3. Choose SSH key option:
   - **Generate new key**: Select key type (Ed25519, RSA, ECDSA)
   - **Paste existing key**: Paste your private key content
4. Press `Tab` to navigate to "Add Host" button
5. Press `Enter` to save

### Editing a Host

1. Select the host with `↑/↓`
2. Press `e` to edit
3. Modify fields as needed
4. Press `Enter` on "Save Changes" to update

### Deleting a Host

1. Select the host with `↑/↓`
2. Press `d` to delete
3. Confirm with `Y` or cancel with `N`

### Searching Hosts

1. Press `/` or `Ctrl+F` to activate search
2. Type to filter hosts by hostname, username, or notes
3. Press `Enter` or `Esc` to exit search mode
4. Press `Esc` again to clear the filter

## Project Structure

```
ssh-manager/
├── main.go              # Entry point
├── app.go               # Bubble Tea application logic
├── models.go            # Data structures
├── ui/
│   ├── styles.go        # Lipgloss styling
│   ├── main.go          # Main view rendering
│   └── modals.go        # Modal components
├── db/
│   └── db.go           # Database interface (stub)
├── ssh/                 # SSH operations (future)
├── crypto/              # Encryption layer (future)
└── README.md
```

## Architecture

### MVP (Current Phase)
- **TUI Framework**: Bubble Tea for reactive UI
- **Styling**: Lipgloss for terminal colors/layout
- **Data Storage**: Hardcoded sample data (for testing)
- **Navigation**: Arrow keys + keyboard shortcuts

### Phase 2 (Planned)
```
┌─────────────────────────────────────────────┐
│  TUI Layer (Bubble Tea)                    │
│  ├── Main screen: host list + details      │
│  ├── Modals: add/edit/delete hosts         │
│  └── Keybindings: arrow keys, shortcuts    │
└──────────────┬──────────────────────────────┘
               │
┌──────────────▼──────────────────────────────┐
│  Business Logic                            │
│  ├── db: host CRUD + encryption/decryption │
│  ├── ssh: key generation + connection      │
│  └── crypto: PBKDF2, Fernet, master key    │
└──────────────┬──────────────────────────────┘
               │
┌──────────────▼──────────────────────────────┐
│  Encrypted Database (~/.ssh-manager/db)    │
│  Table: hosts                              │
│  ├── hostname, username, port              │
│  ├── key_data (Fernet-encrypted blob)      │
│  └── metadata (created_at, last_connected) │
└─────────────────────────────────────────────┘
```

## Security (Planned for Phase 2)

### Encryption Layers
1. **User Master Password** - Single password to unlock everything
2. **PBKDF2 Key Derivation** - 100,000 iterations for password hardening
3. **SQLCipher** - AES-256 encryption for entire database
4. **Fernet Encryption** - Additional per-key symmetric encryption
5. **Secure Temp Files** - Keys written to temp files with chmod 600, deleted after use

### Security Features
- ✅ Private keys never stored unencrypted on disk
- ✅ Keys decrypted only in memory during SSH connection
- ✅ Automatic cleanup of temporary key files
- ✅ No logging of sensitive data
- ✅ Secure random number generation for key creation

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
# Development build
go build -o ssh-manager

# Production build (optimized)
go build -ldflags="-s -w" -o ssh-manager
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Roadmap

### ✅ Phase 1: Project Setup (Complete)
- [x] Go module initialization
- [x] Bubble Tea framework integration
- [x] Project structure

### ✅ Phase 4: Core UI (Complete)
- [x] Two-panel layout
- [x] Host list rendering
- [x] Arrow key navigation
- [x] Search/filter functionality

### ✅ Phase 6: Modals & Forms (Complete)
- [x] Add host modal
- [x] Edit host modal
- [x] Delete confirmation
- [x] Generate vs. Paste key options
- [x] Form validation

### 🚧 Phase 2: Database & Encryption (Next)
- [ ] SQLCipher integration
- [ ] Master password setup
- [ ] PBKDF2 key derivation
- [ ] Fernet per-key encryption
- [ ] Persistent storage

### 📅 Phase 3: SSH Operations (Future)
- [ ] SSH key generation (ssh-keygen wrapper)
- [ ] Secure temp file handling
- [ ] SSH connection subprocess
- [ ] Connection history tracking

### 📅 Phase 5: Polish & Features (Future)
- [ ] Vim keybindings mode
- [ ] Configuration file support
- [ ] Export/import hosts
- [ ] Connection timeout handling
- [ ] SSH config file integration

## Tech Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Components**: [Bubbles](https://github.com/charmbracelet/bubbles)
- **Database** (planned): [SQLCipher](https://github.com/mutecomm/go-sqlcipher)
- **Encryption** (planned): Go crypto standard library

## License

MIT License - see LICENSE file for details

## Acknowledgments

- [Charm](https://charm.sh/) for the amazing TUI libraries
- [SQLCipher](https://www.zetetic.net/sqlcipher/) for encrypted SQLite

## Contact

- GitHub: [@Vansh-Raja](https://github.com/Vansh-Raja)
- Project: [SSHThing](https://github.com/Vansh-Raja/SSHThing)

---

**Note**: This is currently an MVP prototype. Encryption and SSH connection features are planned for Phase 2. Do not use for production systems yet.
