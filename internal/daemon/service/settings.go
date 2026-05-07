package service

import (
	"fmt"
	"sync/atomic"

	"github.com/Vansh-Raja/SSHThing/internal/config"
)

// CfgStore provides thread-safe atomic access to a config.Config value.
// Use atomic.Pointer[config.Config] semantics: get the value, mutate a copy, store.
type CfgStore struct {
	p atomic.Pointer[config.Config]
}

// NewCfgStore creates a CfgStore seeded with cfg.
func NewCfgStore(cfg config.Config) *CfgStore {
	cs := &CfgStore{}
	cs.p.Store(&cfg)
	return cs
}

// Get returns the current config (always non-nil after NewCfgStore).
func (cs *CfgStore) Get() config.Config {
	v := cs.p.Load()
	if v == nil {
		return config.Default()
	}
	return *v
}

// Set atomically replaces the stored config.
func (cs *CfgStore) Set(cfg config.Config) {
	cs.p.Store(&cfg)
}

// Settings provides settings get/set operations backed by CfgStore + config.Save.
type Settings struct {
	Store *CfgStore
}

// Get returns the current configuration as loaded by config.Load().
func (s *Settings) Get() config.Config {
	return s.Store.Get()
}

// Set overwrites the stored config with cfg, persists it to disk via config.Save,
// and hot-reloads the in-memory CfgStore so all services see the new value immediately.
// The renderer is responsible for sending the full Config object; no partial-merge
// is performed here.
func (s *Settings) Set(cfg config.Config) error {
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	s.Store.Set(cfg)
	return nil
}
