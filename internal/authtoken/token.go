package authtoken

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	idBytes        = 10
	secretBytes    = 32
	saltBytes      = 16
	argonTime      = 1
	argonMemoryKiB = 19 * 1024
	argonThreads   = 1
	argonKeyLen    = 32

	payloadAADLabel = "sshthing.authtoken.payload.v1"
	unlockAADLabel  = "sshthing.authtoken.dbunlock.v2"
)

func Parse(raw string) (tokenID string, secret string, err error) {
	raw = strings.TrimSpace(raw)
	parts := strings.SplitN(raw, "_", 3)
	if len(parts) != 3 || parts[0] != TokenPrefix {
		return "", "", fmt.Errorf("invalid token format")
	}
	if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
		return "", "", fmt.Errorf("invalid token format")
	}
	return parts[1], parts[2], nil
}

func CreateToken(name string, grants []HostGrant, dbUnlockSecret string, opts CreateOptions) (string, StoredToken, error) {
	if len(grants) == 0 {
		return "", StoredToken{}, fmt.Errorf("at least one host is required")
	}
	dbUnlockSecret = strings.TrimSpace(dbUnlockSecret)
	if dbUnlockSecret == "" {
		return "", StoredToken{}, fmt.Errorf("missing database unlock secret")
	}
	tokenID, err := randomID()
	if err != nil {
		return "", StoredToken{}, err
	}
	return createTokenWithID(tokenID, name, grants, dbUnlockSecret, opts)
}

func createTokenWithID(tokenID string, name string, grants []HostGrant, dbUnlockSecret string, opts CreateOptions) (string, StoredToken, error) {
	secret, err := randomSecret()
	if err != nil {
		return "", StoredToken{}, err
	}

	hashSalt, err := randomBytes(saltBytes)
	if err != nil {
		return "", StoredToken{}, err
	}
	hash := tokenHash(secret, hashSalt)

	now := time.Now().UTC()
	rec := StoredToken{
		TokenID:     strings.TrimSpace(tokenID),
		Name:        strings.TrimSpace(name),
		Salt:        base64.RawStdEncoding.EncodeToString(hashSalt),
		Hash:        base64.RawStdEncoding.EncodeToString(hash),
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   opts.ExpiresAt,
		MaxUses:     opts.MaxUses,
		SyncEnabled: opts.SyncEnabled,
		Hosts:       make([]StoredTokenHost, 0, len(grants)),
	}

	for _, g := range grants {
		if g.HostID <= 0 {
			return "", StoredToken{}, fmt.Errorf("invalid host id in scope")
		}
		rec.Hosts = append(rec.Hosts, StoredTokenHost{
			HostID:       g.HostID,
			DisplayLabel: strings.TrimSpace(g.DisplayLabel),
		})
	}

	unlockSalt, err := randomBytes(saltBytes)
	if err != nil {
		return "", StoredToken{}, err
	}
	wrapped, err := wrapDBUnlock(secret, dbUnlockSecret, unlockSalt, opts.DevicePepper, opts.BindToDevice)
	if err != nil {
		return "", StoredToken{}, err
	}
	rec.UnlockSalt = base64.RawStdEncoding.EncodeToString(unlockSalt)
	rec.UnlockData = wrapped
	rec.UnlockBound = opts.BindToDevice && len(opts.DevicePepper) > 0

	return fmt.Sprintf("%s_%s_%s", TokenPrefix, rec.TokenID, secret), rec, nil
}

func Verify(raw string, rec StoredToken) (string, bool) {
	tokenID, secret, err := Parse(raw)
	if err != nil || tokenID != rec.TokenID {
		return "", false
	}
	if strings.TrimSpace(rec.Salt) == "" || strings.TrimSpace(rec.Hash) == "" {
		return "", false
	}
	salt, err := base64.RawStdEncoding.DecodeString(rec.Salt)
	if err != nil {
		return "", false
	}
	want, err := base64.RawStdEncoding.DecodeString(rec.Hash)
	if err != nil {
		return "", false
	}
	got := tokenHash(secret, salt)
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return "", false
	}
	return secret, true
}

func (t StoredToken) IsLegacyPayload() bool {
	for _, h := range t.Hosts {
		if strings.TrimSpace(h.Payload) != "" {
			return true
		}
	}
	return false
}

func (t StoredToken) IsUsable() bool {
	if t.DeletedAt != nil || t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && time.Now().UTC().After(*t.ExpiresAt) {
		return false
	}
	if t.MaxUses > 0 && t.UseCount >= t.MaxUses {
		return false
	}
	if t.IsLegacyPayload() {
		return true
	}
	return strings.TrimSpace(t.Salt) != "" && strings.TrimSpace(t.Hash) != "" && strings.TrimSpace(t.UnlockData) != ""
}

func resolveRecord(rawToken string, rec StoredToken, targetLabel string, devicePepper []byte) (ResolveResult, error) {
	secret, ok := Verify(rawToken, rec)
	if !ok {
		return ResolveResult{}, fmt.Errorf("invalid token")
	}
	if rec.DeletedAt != nil {
		return ResolveResult{}, fmt.Errorf("token deleted")
	}
	if rec.RevokedAt != nil {
		return ResolveResult{}, fmt.Errorf("token revoked")
	}
	if rec.ExpiresAt != nil && time.Now().UTC().After(*rec.ExpiresAt) {
		return ResolveResult{}, fmt.Errorf("token expired")
	}
	if rec.MaxUses > 0 && rec.UseCount >= rec.MaxUses {
		return ResolveResult{}, fmt.Errorf("token usage limit reached")
	}

	targetLabel = strings.TrimSpace(targetLabel)
	if targetLabel == "" {
		return ResolveResult{}, fmt.Errorf("target label is required")
	}

	matches := make([]StoredTokenHost, 0, 1)
	for _, h := range rec.Hosts {
		if strings.TrimSpace(h.DisplayLabel) == targetLabel {
			matches = append(matches, h)
		}
	}
	if len(matches) == 0 {
		return ResolveResult{}, fmt.Errorf("target not allowed by token")
	}
	if len(matches) > 1 {
		return ResolveResult{}, fmt.Errorf("target label is ambiguous within token scope")
	}

	h := matches[0]
	result := ResolveResult{TokenID: rec.TokenID, HostID: h.HostID, HostLabel: h.DisplayLabel}

	if strings.TrimSpace(h.Payload) != "" {
		salt, err := base64.RawStdEncoding.DecodeString(h.PayloadSalt)
		if err != nil {
			return ResolveResult{}, fmt.Errorf("invalid token host payload")
		}
		p, err := openLegacyPayload(secret, salt, h.Payload)
		if err != nil {
			return ResolveResult{}, fmt.Errorf("failed to decrypt token host payload")
		}
		if p.HostID != h.HostID {
			return ResolveResult{}, fmt.Errorf("token host payload integrity check failed")
		}
		result.LegacyPayload = &p
		return result, nil
	}

	if strings.TrimSpace(rec.UnlockData) == "" || strings.TrimSpace(rec.UnlockSalt) == "" {
		return ResolveResult{}, fmt.Errorf("token is not active on this device")
	}
	salt, err := base64.RawStdEncoding.DecodeString(rec.UnlockSalt)
	if err != nil {
		return ResolveResult{}, fmt.Errorf("invalid token unlock payload")
	}
	secretValue, err := unwrapDBUnlock(secret, rec.UnlockData, salt, devicePepper, rec.UnlockBound)
	if err != nil {
		if rec.UnlockBound {
			return ResolveResult{}, fmt.Errorf("token is bound to this device and keyring is unavailable")
		}
		return ResolveResult{}, fmt.Errorf("failed to unlock token for execution")
	}
	result.DBUnlockSecret = secretValue
	return result, nil
}

func tokenHash(secret string, salt []byte) []byte {
	return argon2.IDKey([]byte(secret), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
}

func unlockKey(secret string, salt []byte, devicePepper []byte, requirePepper bool) ([]byte, error) {
	if requirePepper && len(devicePepper) == 0 {
		return nil, fmt.Errorf("device pepper required")
	}
	input := []byte(secret)
	if len(devicePepper) > 0 {
		input = append(input, '|')
		input = append(input, devicePepper...)
	}
	return argon2.IDKey(input, salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen), nil
}

func wrapDBUnlock(tokenSecret string, unlockSecret string, salt []byte, devicePepper []byte, bindToDevice bool) (string, error) {
	key, err := unlockKey(tokenSecret, salt, devicePepper, bindToDevice)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(unlockSecret), []byte(unlockAADLabel))
	return base64.RawStdEncoding.EncodeToString(ct), nil
}

func unwrapDBUnlock(tokenSecret string, wrapped string, salt []byte, devicePepper []byte, requirePepper bool) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(wrapped)
	if err != nil {
		return "", err
	}
	key, err := unlockKey(tokenSecret, salt, devicePepper, requirePepper)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce := raw[:gcm.NonceSize()]
	ct := raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, []byte(unlockAADLabel))
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func openLegacyPayload(secret string, salt []byte, sealed string) (ExecPayload, error) {
	raw, err := base64.RawStdEncoding.DecodeString(sealed)
	if err != nil {
		return ExecPayload{}, err
	}
	key := argon2.IDKey([]byte(secret), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	block, err := aes.NewCipher(key)
	if err != nil {
		return ExecPayload{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ExecPayload{}, err
	}
	if len(raw) < gcm.NonceSize() {
		return ExecPayload{}, errors.New("ciphertext too short")
	}
	nonce := raw[:gcm.NonceSize()]
	ct := raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, []byte(payloadAADLabel))
	if err != nil {
		return ExecPayload{}, err
	}
	var payload ExecPayload
	if err := json.Unmarshal(pt, &payload); err != nil {
		return ExecPayload{}, err
	}
	return payload, nil
}

func randomID() (string, error) {
	b, err := randomBytes(idBytes)
	if err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	return strings.ToLower(enc), nil
}

func randomSecret() (string, error) {
	b, err := randomBytes(secretBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}
