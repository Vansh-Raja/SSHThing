# SSH Manager TUI - Master Plan

## 🎯 Project Vision

Build a professional SSH manager TUI application for macOS in Go. This is a **portfolio project** demonstrating full-stack security, UX design, and production-grade software engineering.

## 📋 Project Overview

### What We're Building

A TUI SSH Manager that:
- Stores SSH hosts (hostname, username, port) with encrypted key storage
- **Dual-layer encryption**: SQLCipher (database AES-256) + Fernet (per-key encryption)
- Modern, responsive TUI with **Vim keybindings** using Bubble Tea framework
- Generates SSH keys on-demand, decrypts only in memory, writes to temp files before SSH
- **Single binary distribution** (~12MB)
- Clean, maintainable code architecture

### Core Features (Full Vision)

1. **Secure Storage**
   - Encrypted SQLite database (SQLCipher)
   - Per-key Fernet encryption
   - Master password with PBKDF2 key derivation
   - Keys never stored unencrypted on disk

2. **SSH Key Management**
   - Generate new keys (Ed25519, RSA 4096, ECDSA P-256)
   - Import/paste existing keys
   - Decrypt only in memory
   - Auto-cleanup of temp files

3. **Beautiful TUI**
   - Two-panel layout (list + details)
   - Real-time search/filter
   - Modal dialogs for CRUD operations
   - Vim keybindings (j/k/gg/G/etc.)
   - Responsive design

4. **SSH Operations**
   - Direct SSH connection
   - Automatic key injection
   - Connection history tracking
   - Secure temp file handling

## 🏗️ Architecture

```
┌─────────────────────────────────────────────┐
│  TUI Layer (Bubble Tea)                    │
│  ├── Main screen: host list + details      │
│  ├── Modals: add/edit/delete hosts         │
│  └── Keybindings: j/k, /, Enter, e, d, q  │
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

## 📁 Project Structure

```
ssh-manager/
├── main.go                 # Entry point, app initialization
├── app.go                  # Bubble Tea app model & update/view
├── models.go               # Host struct, messages
├── db/
│   └── db.go              # SQLCipher CRUD + encryption layer
├── ssh/
│   ├── keygen.go          # ssh-keygen wrapper
│   └── connect.go         # SSH connection subprocess
├── crypto/
│   └── crypto.go          # PBKDF2, Fernet encryption
├── ui/
│   ├── main.go            # Main view components
│   ├── modals.go          # Add/edit/delete modals
│   └── styles.go          # Lipgloss styling
├── go.mod
└── README.md
```

## 🗂️ Data Model

### Database Schema (SQLCipher)

**Location:** `~/.ssh-manager/hosts.db`

**Table: hosts**
```sql
CREATE TABLE hosts (
    id INTEGER PRIMARY KEY,
    hostname TEXT UNIQUE NOT NULL,
    username TEXT NOT NULL,
    port INTEGER DEFAULT 22,
    key_data BLOB,              -- Fernet-encrypted private key
    key_type TEXT,              -- "ed25519", "rsa", "ecdsa", "pasted"
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_connected TIMESTAMP,
    notes TEXT
);
```

### Encryption Flow

```
User Master Password
    ↓
PBKDF2 (100k iterations)
    ↓
256-bit Key
    ↓
┌───────────────────┐
│  SQLCipher        │  ← Encrypts entire database
│  (AES-256)        │
└───────────────────┘
    ↓
Database unlocked
    ↓
Per-key Fernet encryption
    ↓
Private keys stored as encrypted blobs
```

## 🎯 Implementation Phases

### ✅ Phase 1: Project Setup (Complete)
- [x] Go module initialization
- [x] Bubble Tea framework integration
- [x] Project structure

### ✅ Phase 4: Core UI (Complete)
- [x] Two-panel layout
- [x] Host list rendering
- [x] Arrow key navigation
- [x] Search/filter functionality
- [x] Spotlight Search overlay

### ✅ Phase 6: Modals & Forms (Complete)
- [x] Add host modal (Responsive & Navigable)
- [x] Edit host modal
- [x] Delete confirmation (Navigable)
- [x] Generate vs. Paste key options
- [x] Form validation
- [x] Robust Keyboard Handling (Arrows/Enter/Space/Vim)

### ✅ UI Polish (Complete)
- [x] Fixed modal cut-off on large fonts
- [x] Fixed modal input styling glitches
- [x] Cleaned up modal layout (Host/Port on one row, Auth selector)
- [x] Fully navigable modal forms (Tab, Shift+Tab, Arrows)
- [x] Robust Auth Method selection (Arrows/H/L)
- [x] Delete Confirmation modal navigation

### ✅ Phase 2: Database & Encryption (Complete)

- [x] SQLCipher integration (Full DB encryption with go-sqlcipher driver)
- [x] Master password setup flow (First-run setup + login flow)
- [x] PBKDF2 key derivation (100k iterations)
- [x] Fernet per-key encryption (AES-GCM authenticated encryption)
- [x] CRUD operations (Create/Read/Update/Delete all implemented)
- [x] Data persistence
- [x] Dual-layer encryption (SQLCipher DB + per-key AES-GCM)

**Files created/modified:**
- `crypto/crypto.go` - Encryption utilities (PBKDF2, AES-GCM)
- `db/db.go` - SQLCipher database with encrypted key storage
- `main.go` - Entry point
- `app.go` - Main application logic, DB integration, Login/Setup handling
- `models.go` - Added `ViewModeLogin`, `ViewModeSetup`
- `ui/styles.go` - Added `LoginBox` style
- `ui/login.go` - Login and Setup screen rendering

### ✅ Phase 3: SSH Operations (Complete)

- [x] SSH key generation (ssh-keygen wrapper for Ed25519, RSA-4096, ECDSA)
- [x] Temp file handling (secure create with 600 permissions, auto-cleanup)
- [x] SSH connection subprocess (using tea.ExecProcess)
- [x] Connection history tracking (last_connected timestamp)
- [x] Key decryption in memory only

**Files created:**
- `ssh/keygen.go` - Key generation with ssh-keygen
- `ssh/connect.go` - SSH connections with secure temp file handling

### 📅 Phase 5: Polish & Features (FUTURE)
**Status:** Planned
**Estimated:** 4-5 hours

- [ ] Vim keybindings mode (toggle)
- [ ] Configuration file support
- [ ] Export/import hosts
- [ ] Connection timeout handling
- [ ] SSH config file integration
- [ ] Better error messages
- [ ] Loading states

## 🔐 Security Architecture

### Encryption Layers

1. **User Master Password**
   - User provides password on startup
   - Minimum 12 characters enforced
   - Never stored, only used for key derivation

2. **PBKDF2 Key Derivation**
   - 100,000 iterations
   - SHA-256 hash
   - Random salt (stored in DB header)
   - Produces 256-bit key

3. **SQLCipher (Database Encryption)**
   - AES-256-CBC encryption
   - Entire database encrypted at rest
   - Unlocked with PBKDF2-derived key
   - `PRAGMA key='...'` to unlock

4. **Fernet (Per-Key Encryption)**
   - Additional symmetric encryption per SSH key
   - Prevents key exposure even if DB is unlocked
   - Uses derived key from master password
   - Base64-encoded encrypted blobs

5. **Secure Temp Files**
   - Keys written to `/tmp/ssh_manager_<uuid>`
   - `chmod 600` (owner read/write only)
   - Deleted immediately after SSH exits
   - Never written to disk unencrypted except temp

### Security Flow

```
Startup
  ↓
User enters master password
  ↓
PBKDF2 derive key
  ↓
Unlock SQLCipher DB
  ↓
User selects host
  ↓
Fetch encrypted key blob from DB
  ↓
Decrypt with Fernet (in memory only)
  ↓
Write to temp file with chmod 600
  ↓
ssh -i /tmp/ssh_manager_<uuid> user@host
  ↓
Delete temp file
```

## 🎨 UI/UX Design

### Main Screen Layout

```
┌──────────────────────────────────────────────┐
│ 🔐 SSH Manager                   [?] Help    │  ← Header
├────────────────┬─────────────────────────────┤
│                │                             │
│  HOST LIST     │   DETAILS PANEL            │
│  ──────────    │   ─────────────────────    │
│ ⚡ web-prod-1  │   Hostname: web-prod-1    │
│   db-server    │   Username: ec2-user      │
│   staging      │   Port: 22                │
│                │   Status: Ready ✓         │
│  Filter: /___  │   Last SSH: 2h ago        │
│                │                             │
├────────────────┴─────────────────────────────┤
│ [↑/↓] Navigate [Enter] Connect [?] Help      │  ← Footer
└──────────────────────────────────────────────┘
```

### Keybindings (Current - Standard Mode)

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate list |
| `Ctrl+U/D` | Page up/down |
| `Home/End` | Jump to top/bottom |
| `Enter` | Connect to host |
| `a` | Add new host |
| `e` | Edit host |
| `d` | Delete host |
| `/` | Search/filter |
| `?` | Help |
| `q` or `Ctrl+C` | Quit |

### Keybindings (Future - Vim Mode)

| Key | Action |
|-----|--------|
| `j/k` | Navigate list |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `Ctrl+D/U` | Page down/up |
| `/` | Search |
| `n/N` | Next/previous match |
| `i` | Add host (insert) |
| `e` | Edit host |
| `dd` | Delete host |
| `:q` | Quit |

## 🎯 Success Criteria

### Technical Goals

- ✅ Smooth TUI (no lag, fast startup)
- ✅ Dual-layer encryption working (SQLCipher + AES-GCM)
- ✅ Keys decrypt in memory only
- ✅ Full CRUD on hosts
- ✅ SSH key gen + connection working
- 🚧 Vim keybindings functional (basic support, toggle pending)
- ✅ Graceful error handling
- ✅ Single binary distributable (~9MB)
- ✅ Clean, readable Go code
- ✅ Unit tests for core logic

### Portfolio Talking Points

1. **Architecture**
   - "I built a layered architecture with clear separation between UI, business logic, and data"
   - "Dual-layer encryption with SQLCipher and per-key Fernet encryption"
   - "PBKDF2 key derivation with 100k iterations for password hardening"

2. **Go Expertise**
   - "Chose Go for single binary distribution, excellent TUI libraries, and performance"
   - "Used Bubble Tea framework for reactive UI with Elm architecture"
   - "Implemented interfaces for testability and clean abstractions"

3. **Security Design**
   - "Private keys never touch disk unencrypted except secure temp files"
   - "Temp file handling with chmod 600 and auto-cleanup"
   - "Memory-only decryption with immediate cleanup"

4. **UX Focus**
   - "Responsive design that adapts to terminal size and text scaling"
   - "Implemented real-time search and intuitive keyboard navigation"
   - "Adaptive layout system for accessibility (large text support)"

5. **Testing & Quality**
   - "Wrote comprehensive unit tests covering validation and core logic"
   - "Performance optimized - handles thousands of hosts smoothly"
   - "Production-ready error handling and edge case management"

## 🔨 Tech Stack

### Core
- **Language:** Go 1.21+
- **TUI Framework:** Bubble Tea (reactive, Elm-architecture)
- **Styling:** Lipgloss (terminal styling DSL)
- **Components:** Bubbles (reusable TUI components)

### Security
- **Database:** SQLCipher (AES-256 encrypted SQLite)
- **Key Derivation:** PBKDF2 (100k iterations, SHA-256)
- **Symmetric Encryption:** Fernet (AES-128-CBC + HMAC)
- **Random:** crypto/rand (cryptographically secure)

### SSH
- **Key Generation:** os/exec + ssh-keygen
- **Connections:** os/exec + ssh command
- **Algorithms:** Ed25519 (default), RSA 4096, ECDSA P-256

## 📊 Current Status (v0.3.0)

### ✅ Completed
- Project structure and Go setup
- Beautiful two-panel TUI
- Responsive modal system
- Arrow key navigation
- Search/filter functionality
- Add/Edit/Delete modals
- Form validation
- Adaptive layout for large text
- Unit tests for core logic
- Comprehensive documentation
- Spotlight Search overlay
- Robust Key Handling
- SQLCipher full database encryption
- Dual-layer encryption (DB + per-key AES-GCM)
- Master password setup & login flow
- SSH key generation (Ed25519, RSA, ECDSA)
- SSH connections with secure temp file handling
- Connection history tracking

### 📅 Next Up (Phase 5)
- Vim keybindings mode toggle
- Configuration file support
- Export/import hosts
- SSH config file integration

## 🎓 Learning Objectives

### For This Project

1. **Go Proficiency**
   - Interfaces and composition
   - Error handling patterns
   - Goroutines for async operations
   - Standard library deep dive

2. **TUI Development**
   - Bubble Tea architecture
   - Terminal rendering optimization
   - Responsive design in terminals
   - Keyboard handling

3. **Security Engineering**
   - Encryption at rest
   - Key derivation functions
   - Secure memory handling
   - Threat modeling

4. **Software Architecture**
   - Clean architecture principles
   - Dependency injection
   - Testable code design
   - Interface-based programming

## 🚀 Distribution Plan

### Build Process

```bash
# Development build
go build -o ssh-manager

# Production build (optimized)
go build -ldflags="-s -w" -o ssh-manager

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o ssh-manager-linux
GOOS=darwin GOARCH=arm64 go build -o ssh-manager-macos-arm64
GOOS=windows GOARCH=amd64 go build -o ssh-manager.exe
```

### Installation

```bash
# Manual install
cp ssh-manager /usr/local/bin/

# Homebrew (future)
brew tap Vansh-Raja/ssh-manager
brew install ssh-manager
```

## 📝 Documentation Plan

### User Documentation
- [x] README.md - Overview and quick start
- [x] QUICKSTART.md - Step-by-step tutorial
- [x] CHANGELOG.md - Version history

### Technical Documentation
- [x] PLAN.md - This file (master plan)
- [x] RESPONSIVE_FIX.md - Responsive design solution
- [x] CENTERING_FIX.md - Modal centering solution
- [x] OVERFLOW_FIX.md - Overflow handling solution
- [ ] ENCRYPTION.md - Encryption architecture
- [ ] TESTING.md - Testing strategy
- [ ] CONTRIBUTING.md - Contribution guidelines

## 🎯 Timeline

| Phase | Duration | Status |
|-------|----------|--------|
| **Phase 1** (Setup) | 3h | ✅ Complete |
| **Phase 4** (Core UI) | 5h | ✅ Complete |
| **Phase 6** (Modals) | 6h | ✅ Complete |
| **UI Polish** | 4h | ✅ Complete |
| **Phase 2** (Encryption) | 7h | ✅ Complete |
| **Phase 3** (SSH Ops) | 4h | ✅ Complete |
| **Phase 5** (Polish) | 5h | 📅 Planned |
| **Testing** | 3h | 📅 Planned |
| **Documentation** | 2h | 📅 Planned |
| **Total** | ~39h | ~90% Complete |

## 🎉 Milestone Achievements

- ✅ **Milestone 1:** Working TUI with hardcoded data
- ✅ **Milestone 2:** Complete CRUD UI with modals
- ✅ **Milestone 3:** Responsive design working
- ✅ **Milestone 4:** Frontend complete & polished
- ✅ **Milestone 5:** Persistent encrypted storage
- ✅ **Milestone 6:** SSH connections working
- 🎯 **Milestone 7:** Production-ready v1.0

---

**Last Updated:** 2025-12-13
**Current Version:** 0.3.0 (Core Features Complete)
**Next Target:** Phase 5: Polish & Features

