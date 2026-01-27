# Building SSHThing on Windows

SSHThing uses SQLCipher for encrypted database storage, which requires CGO (Go's C interop). This guide walks through setting up a Windows build environment.

## Prerequisites

### 1. Go 1.25+

Download and install Go from [go.dev/dl](https://go.dev/dl/).

Verify installation:
```powershell
go version
```

### 2. MSYS2 + MinGW-w64

SSHThing requires a C compiler for CGO. We use MSYS2 with MinGW-w64.

1. Download MSYS2 from [msys2.org](https://www.msys2.org/)
2. Run the installer (default location: `C:\msys64`)
3. Open "MSYS2 MINGW64" from the Start Menu
4. Update the package database:
   ```bash
   pacman -Syu
   ```
5. If prompted, close the terminal and reopen "MSYS2 MINGW64"
6. Install required packages:
   ```bash
   pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-pkg-config mingw-w64-x86_64-sqlcipher
   ```

### 3. OpenSSH Client

SSHThing requires `ssh`, `sftp`, and `ssh-keygen` commands.

**Option A: Windows Optional Feature (Recommended)**
1. Open Settings > Apps > Optional Features
2. Click "Add a feature"
3. Search for "OpenSSH Client" and install it

**Option B: Using winget**
```powershell
winget install Microsoft.OpenSSH.Client
```

Verify installation:
```powershell
ssh -V
sftp -V
ssh-keygen -V
```

## Building

### Option 1: Using MSYS2 Terminal (Recommended)

Open "MSYS2 MINGW64" terminal and navigate to the project:

```bash
cd /d/Code/SSHThing  # Adjust path as needed

# Build
CGO_ENABLED=1 go build -o sshthing.exe ./cmd/sshthing

# Run
./sshthing.exe
```

### Option 2: Using PowerShell/CMD

First, add MinGW to your PATH:

```powershell
$env:PATH = "C:\msys64\mingw64\bin;$env:PATH"
$env:CGO_ENABLED = "1"
$env:CC = "gcc"

# Build
go build -o sshthing.exe ./cmd/sshthing

# Run
.\sshthing.exe
```

### Option 3: Persistent Environment Variables

To avoid setting PATH each time:

1. Open System Properties > Environment Variables
2. Under "User variables", edit `Path`
3. Add `C:\msys64\mingw64\bin`
4. Create new variable `CGO_ENABLED` = `1`
5. Restart your terminal

## Troubleshooting

### "gcc not found"

Ensure `C:\msys64\mingw64\bin` is in your PATH:
```powershell
$env:PATH = "C:\msys64\mingw64\bin;$env:PATH"
```

### "sqlcipher.h not found"

Install SQLCipher in MSYS2:
```bash
pacman -S mingw-w64-x86_64-sqlcipher
```

### "undefined reference to sqlite3..."

This usually means the SQLCipher library wasn't linked. Ensure you're using the MINGW64 environment, not MSYS or UCRT64.

### CGO_ENABLED errors

Make sure CGO is enabled:
```powershell
$env:CGO_ENABLED = "1"
go env CGO_ENABLED  # Should print "1"
```

## Release Build

For a smaller, optimized binary:

```bash
CGO_ENABLED=1 go build -ldflags="-s -w" -o sshthing.exe ./cmd/sshthing
```

The `-s -w` flags strip debug symbols and DWARF info, reducing binary size.

## Cross-Compiling from macOS/Linux

If you prefer to build Windows binaries from a Unix system:

```bash
# Install mingw-w64 cross-compiler
# macOS: brew install mingw-w64
# Ubuntu: sudo apt install gcc-mingw-w64-x86-64

# Cross-compile
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  CC=x86_64-w64-mingw32-gcc \
  go build -o sshthing.exe ./cmd/sshthing
```

Note: Cross-compiling with CGO is more complex and may require additional setup for SQLCipher dependencies.

## Data Locations

On Windows, SSHThing stores data in:

| Data | Location |
|------|----------|
| Database | `%APPDATA%\sshthing\hosts.db` |
| Config | `%APPDATA%\sshthing\config.json` |
| Sync repo | `%APPDATA%\sshthing\sync\` |

Example: `C:\Users\YourName\AppData\Roaming\sshthing\hosts.db`
