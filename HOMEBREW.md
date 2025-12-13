# Homebrew (Tap)

This repo includes a Homebrew **tap** formula at `Formula/sshthing.rb`.

## Publishing Steps (One-Time)

1. Create a new GitHub repository named `homebrew-tap` under your account.
2. Clone it locally, and add the formula:
   - Put `sshthing.rb` under `Formula/sshthing.rb`
3. Push to GitHub.

## Install (Users)

```bash
brew tap Vansh-Raja/tap
brew install sshthing
```

## Notes

- The formula currently uses `head` (builds from `main`). For a stable install, create tagged releases and switch the formula to `url` + `sha256`.
- SQLCipher is required at build time (`depends_on "sqlcipher"`). This uses CGO.
- Finder mounts are optional and require installing FUSE-T + SSHFS (`fuse-t` and `fuse-t-sshfs` casks).

