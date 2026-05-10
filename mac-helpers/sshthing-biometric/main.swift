// sshthing-biometric — tiny CLI that wraps macOS Keychain + LocalAuthentication
// so the daemon can store the SSHThing master password protected by Touch ID.
//
// Subcommands:
//   available                          exit 0 if Touch ID is supported & enrolled, else 1
//   store    --service S --account A   read secret from stdin, store in keychain
//   fetch    --service S --account A [--reason R]   triggers Touch ID prompt then returns secret
//   forget   --service S --account A   delete the keychain item
//
// Exit codes:
//   0  success
//   1  Touch ID unavailable / not enrolled / cancelled
//   2  user authentication failed (e.g. fingerprint mismatch retries exhausted)
//   3  no item found
//   4  other / I/O error
//   64 usage error
//
// Implementation note: we deliberately do NOT use SecAccessControl with
// .biometryCurrentSet. That requires the modern data-protection keychain,
// which on macOS demands a `keychain-access-groups` entitlement signed
// with a real Apple Developer Team ID. Ad-hoc signed binaries can't
// satisfy that requirement (kernel kills the process at launch).
//
// Instead we store the secret in the legacy file-based keychain (already
// encrypted at rest with the user's macOS account password) and gate
// fetches via LAContext.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics).
// Threat model is roughly equivalent: another app on the same logged-in
// session would need to bypass Touch ID + the OS keychain to extract the
// secret, and our `fetch` won't return it without a successful biometric
// authentication.
//
// All status output goes to stderr. Only `fetch` writes the secret to stdout.

import Foundation
import Security
import LocalAuthentication

@inline(__always)
func errprint(_ s: String) {
    if let data = (s + "\n").data(using: .utf8) { FileHandle.standardError.write(data) }
}

func usage() -> Never {
    errprint("""
    sshthing-biometric — Touch ID helper

      available
      store  --service <s> --account <a>
      fetch  --service <s> --account <a> [--reason <r>]
      forget --service <s> --account <a>
    """)
    exit(64)
}

// ── arg parsing ─────────────────────────────────────────────────────────
struct Args {
    var service: String?
    var account: String?
    var reason: String?
}

func parseArgs(_ argv: ArraySlice<String>) -> Args {
    var args = Args()
    var i = argv.startIndex
    while i < argv.endIndex {
        let a = argv[i]
        i = argv.index(after: i)
        guard i < argv.endIndex else { errprint("missing value for \(a)"); exit(64) }
        let v = argv[i]
        i = argv.index(after: i)
        switch a {
        case "--service": args.service = v
        case "--account": args.account = v
        case "--reason":  args.reason = v
        default: errprint("unknown flag \(a)"); exit(64)
        }
    }
    return args
}

// ── available ───────────────────────────────────────────────────────────
func cmdAvailable() -> Int32 {
    let ctx = LAContext()
    var err: NSError?
    let ok = ctx.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &err)
    if !ok {
        if let e = err {
            errprint("touch id unavailable: \(e.localizedDescription)")
        } else {
            errprint("touch id unavailable")
        }
        return 1
    }
    return 0
}

// ── store ───────────────────────────────────────────────────────────────
func cmdStore(_ args: Args) -> Int32 {
    guard let service = args.service, let account = args.account else { usage() }

    let secret = FileHandle.standardInput.availableData
    if secret.isEmpty {
        errprint("no secret on stdin")
        return 64
    }

    // Idempotent: delete first.
    let deleteQuery: [String: Any] = [
        kSecClass as String:        kSecClassGenericPassword,
        kSecAttrService as String:  service,
        kSecAttrAccount as String:  account,
    ]
    SecItemDelete(deleteQuery as CFDictionary)

    let addQuery: [String: Any] = [
        kSecClass as String:           kSecClassGenericPassword,
        kSecAttrService as String:     service,
        kSecAttrAccount as String:     account,
        kSecValueData as String:       secret,
        kSecAttrAccessible as String:  kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
        kSecAttrSynchronizable as String: false,
    ]
    let status = SecItemAdd(addQuery as CFDictionary, nil)
    if status != errSecSuccess {
        errprint("SecItemAdd failed: \(status)")
        return 4
    }
    return 0
}

// ── fetch ───────────────────────────────────────────────────────────────
//
// Two-step flow:
//   1. LAContext.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics)
//      shows the Touch ID dialog. If the user cancels or fails, we exit
//      without ever reading the keychain.
//   2. On success, read the keychain item normally and write its bytes
//      to stdout.
//
// This gives us biometric gating without needing the data-protection
// keychain (which ad-hoc signing can't access).
func cmdFetch(_ args: Args) -> Int32 {
    guard let service = args.service, let account = args.account else { usage() }
    let reason = args.reason ?? "Unlock SSHThing"

    let context = LAContext()
    let semaphore = DispatchSemaphore(value: 0)
    var authOK = false
    var authError: Error?

    context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: reason) { ok, err in
        authOK = ok
        authError = err
        semaphore.signal()
    }
    // Block until the dialog resolves. There's no sane timeout here —
    // matches the user's natural cadence.
    semaphore.wait()

    if !authOK {
        if let e = authError as NSError? {
            // LAError codes: see LAError.h
            // userCancel = -2, systemCancel = -4, appCancel = -9, userFallback = -3
            if e.code == LAError.userCancel.rawValue
                || e.code == LAError.appCancel.rawValue
                || e.code == LAError.systemCancel.rawValue
                || e.code == LAError.userFallback.rawValue {
                errprint("user cancelled")
                return 1
            }
            if e.code == LAError.authenticationFailed.rawValue {
                errprint("authentication failed")
                return 2
            }
            errprint("auth error: \(e.localizedDescription)")
        } else {
            errprint("auth failed without error")
        }
        return 2
    }

    // Touch ID succeeded. Now retrieve the keychain item.
    let query: [String: Any] = [
        kSecClass as String:       kSecClassGenericPassword,
        kSecAttrService as String: service,
        kSecAttrAccount as String: account,
        kSecReturnData as String:  true,
        kSecMatchLimit as String:  kSecMatchLimitOne,
    ]
    var item: CFTypeRef?
    let status = SecItemCopyMatching(query as CFDictionary, &item)
    switch status {
    case errSecSuccess:
        guard let data = item as? Data else {
            errprint("no data returned")
            return 4
        }
        FileHandle.standardOutput.write(data)
        return 0
    case errSecItemNotFound:
        errprint("not found")
        return 3
    default:
        errprint("SecItemCopyMatching failed: \(status)")
        return 4
    }
}

// ── forget ──────────────────────────────────────────────────────────────
func cmdForget(_ args: Args) -> Int32 {
    guard let service = args.service, let account = args.account else { usage() }

    let query: [String: Any] = [
        kSecClass as String:       kSecClassGenericPassword,
        kSecAttrService as String: service,
        kSecAttrAccount as String: account,
    ]
    let status = SecItemDelete(query as CFDictionary)
    switch status {
    case errSecSuccess, errSecItemNotFound:
        return 0
    default:
        errprint("SecItemDelete failed: \(status)")
        return 4
    }
}

// ── main ────────────────────────────────────────────────────────────────
let args = CommandLine.arguments
if args.count < 2 { usage() }
let cmd = args[1]
let rest = args.dropFirst(2)
switch cmd {
case "available": exit(cmdAvailable())
case "store":     exit(cmdStore(parseArgs(rest)))
case "fetch":     exit(cmdFetch(parseArgs(rest)))
case "forget":    exit(cmdForget(parseArgs(rest)))
default:          usage()
}
