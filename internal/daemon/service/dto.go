// Package service provides thin service wrappers around internal/* packages,
// exposing types safe to send over the RPC wire (no encrypted blobs, no master keys).
package service

import (
	"strconv"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/db"
)

// HostSummary is the safe over-the-wire representation of a host.
// It omits KeyData (the encrypted blob) and the master key entirely.
// IDs are stringified ints for forward-compat with future UUID migration.
type HostSummary struct {
	ID              string     `json:"id"`
	SyncID          string     `json:"syncId"`
	Label           string     `json:"label"`
	Hostname        string     `json:"hostname"`
	Username        string     `json:"username"`
	Port            int        `json:"port"`
	Group           string     `json:"group"`
	Tags            []string   `json:"tags"`
	LastConnectedAt *time.Time `json:"lastConnectedAt"`
	// AuthMode is "key" | "password" | "none" derived from KeyType.
	AuthMode string `json:"authMode"`
}

// hostSummaryFromModel converts a db.HostModel to a HostSummary.
// Ref: internal/db/db.go HostModel struct.
func hostSummaryFromModel(h db.HostModel) HostSummary {
	authMode := "none"
	switch h.KeyType {
	case "password":
		authMode = "password"
	default:
		if h.KeyData != "" {
			authMode = "key"
		}
	}
	tags := h.Tags
	if tags == nil {
		tags = []string{}
	}
	return HostSummary{
		ID:              strconv.Itoa(h.ID),
		SyncID:          h.SyncID,
		Label:           h.Label,
		Hostname:        h.Hostname,
		Username:        h.Username,
		Port:            h.Port,
		Group:           h.GroupName,
		Tags:            tags,
		LastConnectedAt: h.LastConnected,
		AuthMode:        authMode,
	}
}
