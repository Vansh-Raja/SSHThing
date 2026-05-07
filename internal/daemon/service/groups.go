package service

import (
	"fmt"
	"strings"
)

// Groups provides group management operations on top of an unlocked Vault.
type Groups struct {
	Vault *Vault
}

// List returns all group names in the vault.
// Mirrors: internal/app/backend.go loadGroups → store.GetGroups().
func (g *Groups) List() ([]string, error) {
	store := g.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}
	groups, err := store.GetGroups()
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}
	if groups == nil {
		return []string{}, nil
	}
	return groups, nil
}

// Create creates (or no-ops if already exists) a group with the given name.
// Mirrors: internal/app/handlers.go handleGroupInputKeys → store.UpsertGroup.
func (g *Groups) Create(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	if strings.EqualFold(name, "Ungrouped") {
		return fmt.Errorf("'Ungrouped' is reserved")
	}
	store := g.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	return store.UpsertGroup(name)
}

// Rename renames a group.
// Mirrors: internal/app/handlers.go handleGroupInputKeys → store.RenameGroup.
func (g *Groups) Rename(oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("group names cannot be empty")
	}
	if strings.EqualFold(newName, "Ungrouped") {
		return fmt.Errorf("'Ungrouped' is reserved")
	}
	store := g.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	return store.RenameGroup(oldName, newName)
}

// Delete removes a group.
// Mirrors: internal/app/handlers.go handleDeleteGroupKeys → store.DeleteGroup.
func (g *Groups) Delete(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	store := g.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	return store.DeleteGroup(name)
}
