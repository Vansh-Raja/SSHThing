package personalsync

type KDF struct {
	Name       string `json:"name"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
}

type VaultSummary struct {
	VaultID           string `json:"vaultId"`
	SchemaVersion     int    `json:"schemaVersion"`
	EncryptionVersion string `json:"encryptionVersion"`
	KDF               KDF    `json:"kdf"`
	UpdatedAt         int64  `json:"updatedAt"`
	LatestRevision    int64  `json:"latestRevision,omitempty"`
}

type VaultItem struct {
	ItemType      string `json:"itemType"`
	SyncID        string `json:"syncId"`
	Ciphertext    string `json:"ciphertext"`
	Nonce         string `json:"nonce"`
	UpdatedAt     int64  `json:"updatedAt"`
	DeletedAt     *int64 `json:"deletedAt,omitempty"`
	SchemaVersion int    `json:"schemaVersion"`
	ItemRevision  int64  `json:"itemRevision,omitempty"`
	BaseRevision  string `json:"baseRevision,omitempty"`
}

type ListItemsResponse struct {
	Revision       string      `json:"revision"`
	LatestRevision int64       `json:"latestRevision,omitempty"`
	Items          []VaultItem `json:"items"`
}

type ChangesResponse = ListItemsResponse

type UpsertRequest struct {
	BaseRevision string      `json:"baseRevision,omitempty"`
	DeviceID     string      `json:"deviceId"`
	Force        bool        `json:"force,omitempty"`
	Items        []VaultItem `json:"items"`
}

type Conflict struct {
	ItemType string `json:"itemType"`
	SyncID   string `json:"syncId"`
	RemoteAt int64  `json:"remoteAt"`
	LocalAt  int64  `json:"localAt"`
}

type UpsertResponse struct {
	OK             bool       `json:"ok"`
	Revision       string     `json:"revision"`
	LatestRevision int64      `json:"latestRevision,omitempty"`
	Accepted       int        `json:"accepted,omitempty"`
	Conflicts      []Conflict `json:"conflicts,omitempty"`
}

type DeviceSeenRequest struct {
	DeviceID           string `json:"deviceId"`
	DeviceName         string `json:"deviceName,omitempty"`
	LastPulledRevision int64  `json:"lastPulledRevision,omitempty"`
	LastPushedRevision int64  `json:"lastPushedRevision,omitempty"`
}

type SyncEventRequest struct {
	DeviceID  string `json:"deviceId,omitempty"`
	Source    string `json:"source"`
	Action    string `json:"action"`
	ItemType  string `json:"itemType,omitempty"`
	ItemCount int    `json:"itemCount,omitempty"`
}
