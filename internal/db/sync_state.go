package db

import (
	"database/sql"
	"strings"
	"time"
)

func (s *Store) UpsertSyncItemState(state SyncItemState) error {
	if s == nil || s.db == nil {
		return sql.ErrConnDone
	}
	itemType := strings.TrimSpace(state.ItemType)
	syncID := strings.TrimSpace(state.SyncID)
	if itemType == "" || syncID == "" {
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO sync_items (
			item_type, sync_id, local_revision, remote_revision, local_updated_at,
			remote_updated_at, dirty, deleted, last_pushed_at, last_pulled_at, last_error
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(item_type, sync_id) DO UPDATE SET
			local_revision = excluded.local_revision,
			remote_revision = excluded.remote_revision,
			local_updated_at = excluded.local_updated_at,
			remote_updated_at = excluded.remote_updated_at,
			dirty = excluded.dirty,
			deleted = excluded.deleted,
			last_pushed_at = excluded.last_pushed_at,
			last_pulled_at = excluded.last_pulled_at,
			last_error = excluded.last_error
	`, itemType, syncID, state.LocalRevision, state.RemoteRevision, timePtrValue(state.LocalUpdatedAt),
		timePtrValue(state.RemoteUpdatedAt), boolInt(state.Dirty), boolInt(state.Deleted),
		timePtrValue(state.LastPushedAt), timePtrValue(state.LastPulledAt), state.LastError)
	return err
}

func (s *Store) GetSyncItemState(itemType, syncID string) (SyncItemState, bool, error) {
	if s == nil || s.db == nil {
		return SyncItemState{}, false, sql.ErrConnDone
	}
	var state SyncItemState
	var localUpdated, remoteUpdated, lastPushed, lastPulled sql.NullTime
	var dirty, deleted int
	err := s.db.QueryRow(`
		SELECT item_type, sync_id, COALESCE(local_revision, ''), COALESCE(remote_revision, ''),
			local_updated_at, remote_updated_at, dirty, deleted, last_pushed_at, last_pulled_at,
			COALESCE(last_error, '')
		FROM sync_items
		WHERE item_type = ? AND sync_id = ?
	`, strings.TrimSpace(itemType), strings.TrimSpace(syncID)).Scan(
		&state.ItemType, &state.SyncID, &state.LocalRevision, &state.RemoteRevision,
		&localUpdated, &remoteUpdated, &dirty, &deleted, &lastPushed, &lastPulled, &state.LastError,
	)
	if err == sql.ErrNoRows {
		return SyncItemState{}, false, nil
	}
	if err != nil {
		return SyncItemState{}, false, err
	}
	state.LocalUpdatedAt = nullTimePtr(localUpdated)
	state.RemoteUpdatedAt = nullTimePtr(remoteUpdated)
	state.LastPushedAt = nullTimePtr(lastPushed)
	state.LastPulledAt = nullTimePtr(lastPulled)
	state.Dirty = dirty != 0
	state.Deleted = deleted != 0
	return state, true, nil
}

func (s *Store) ListDirtySyncItems() ([]SyncItemState, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrConnDone
	}
	rows, err := s.db.Query(`
		SELECT item_type, sync_id, COALESCE(local_revision, ''), COALESCE(remote_revision, ''),
			local_updated_at, remote_updated_at, dirty, deleted, last_pushed_at, last_pulled_at,
			COALESCE(last_error, '')
		FROM sync_items
		WHERE dirty = 1
		ORDER BY item_type, sync_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SyncItemState
	for rows.Next() {
		var state SyncItemState
		var localUpdated, remoteUpdated, lastPushed, lastPulled sql.NullTime
		var dirty, deleted int
		if err := rows.Scan(&state.ItemType, &state.SyncID, &state.LocalRevision, &state.RemoteRevision,
			&localUpdated, &remoteUpdated, &dirty, &deleted, &lastPushed, &lastPulled, &state.LastError); err != nil {
			return nil, err
		}
		state.LocalUpdatedAt = nullTimePtr(localUpdated)
		state.RemoteUpdatedAt = nullTimePtr(remoteUpdated)
		state.LastPushedAt = nullTimePtr(lastPushed)
		state.LastPulledAt = nullTimePtr(lastPulled)
		state.Dirty = dirty != 0
		state.Deleted = deleted != 0
		out = append(out, state)
	}
	return out, rows.Err()
}

func (s *Store) MarkSyncItemDirty(itemType, syncID string, deleted bool) error {
	now := time.Now().UTC()
	return s.UpsertSyncItemState(SyncItemState{
		ItemType:       itemType,
		SyncID:         syncID,
		LocalUpdatedAt: &now,
		Dirty:          true,
		Deleted:        deleted,
	})
}

func (s *Store) ClearSyncItemDirty(itemType, syncID, remoteRevision string, syncedAt time.Time) error {
	if s == nil || s.db == nil {
		return sql.ErrConnDone
	}
	_, err := s.db.Exec(`
		UPDATE sync_items
		SET dirty = 0,
			remote_revision = ?,
			last_pushed_at = ?,
			last_error = ''
		WHERE item_type = ? AND sync_id = ?
	`, strings.TrimSpace(remoteRevision), syncedAt.UTC(), strings.TrimSpace(itemType), strings.TrimSpace(syncID))
	return err
}

func (s *Store) UpsertSyncProviderState(state SyncProviderState) error {
	if s == nil || s.db == nil {
		return sql.ErrConnDone
	}
	provider := strings.TrimSpace(state.Provider)
	if provider == "" {
		return nil
	}
	_, err := s.db.Exec(`
		INSERT INTO sync_state (
			provider, vault_id, last_pulled_revision, last_pushed_revision, last_sync_at, last_error
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider) DO UPDATE SET
			vault_id = excluded.vault_id,
			last_pulled_revision = excluded.last_pulled_revision,
			last_pushed_revision = excluded.last_pushed_revision,
			last_sync_at = excluded.last_sync_at,
			last_error = excluded.last_error
	`, provider, state.VaultID, state.LastPulledRevision, state.LastPushedRevision, timePtrValue(state.LastSyncAt), state.LastError)
	return err
}

func (s *Store) GetSyncProviderState(provider string) (SyncProviderState, bool, error) {
	if s == nil || s.db == nil {
		return SyncProviderState{}, false, sql.ErrConnDone
	}
	var state SyncProviderState
	var lastSync sql.NullTime
	err := s.db.QueryRow(`
		SELECT provider, COALESCE(vault_id, ''), COALESCE(last_pulled_revision, ''),
			COALESCE(last_pushed_revision, ''), last_sync_at, COALESCE(last_error, '')
		FROM sync_state
		WHERE provider = ?
	`, strings.TrimSpace(provider)).Scan(&state.Provider, &state.VaultID, &state.LastPulledRevision,
		&state.LastPushedRevision, &lastSync, &state.LastError)
	if err == sql.ErrNoRows {
		return SyncProviderState{}, false, nil
	}
	if err != nil {
		return SyncProviderState{}, false, err
	}
	state.LastSyncAt = nullTimePtr(lastSync)
	return state, true, nil
}

func (s *Store) EnqueueSyncOutbox(item SyncOutboxItem) (int64, error) {
	if s == nil || s.db == nil {
		return 0, sql.ErrConnDone
	}
	itemType := strings.TrimSpace(item.ItemType)
	syncID := strings.TrimSpace(item.SyncID)
	operation := strings.TrimSpace(item.Operation)
	if itemType == "" || syncID == "" || operation == "" {
		return 0, nil
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	res, err := s.db.Exec(`
		INSERT INTO sync_outbox (
			item_type, sync_id, operation, base_revision, encrypted_payload, created_at, attempts, last_error
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, itemType, syncID, operation, item.BaseRevision, item.EncryptedPayload, createdAt.UTC(), item.Attempts, item.LastError)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListSyncOutbox(limit int) ([]SyncOutboxItem, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrConnDone
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT id, item_type, sync_id, operation, COALESCE(base_revision, ''),
			COALESCE(encrypted_payload, ''), created_at, attempts, COALESCE(last_error, '')
		FROM sync_outbox
		ORDER BY created_at, id
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SyncOutboxItem
	for rows.Next() {
		var item SyncOutboxItem
		if err := rows.Scan(&item.ID, &item.ItemType, &item.SyncID, &item.Operation, &item.BaseRevision,
			&item.EncryptedPayload, &item.CreatedAt, &item.Attempts, &item.LastError); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSyncOutboxItem(id int64) error {
	if s == nil || s.db == nil {
		return sql.ErrConnDone
	}
	_, err := s.db.Exec(`DELETE FROM sync_outbox WHERE id = ?`, id)
	return err
}

func (s *Store) MarkSyncOutboxError(id int64, message string) error {
	if s == nil || s.db == nil {
		return sql.ErrConnDone
	}
	_, err := s.db.Exec(`UPDATE sync_outbox SET attempts = attempts + 1, last_error = ? WHERE id = ?`, strings.TrimSpace(message), id)
	return err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func timePtrValue(v *time.Time) any {
	if v == nil || v.IsZero() {
		return nil
	}
	return v.UTC()
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
