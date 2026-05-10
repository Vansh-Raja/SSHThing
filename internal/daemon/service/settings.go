package service

import (
	"sync"
	"sync/atomic"

	"github.com/Vansh-Raja/SSHThing/internal/config"
)

// CfgStore provides thread-safe access to a config.Config value.
// Reads are lock-free via atomic.Pointer; writes that depend on the prior
// value (read-modify-write) MUST go through Mutate so they serialise with
// each other and with config.Save (whose `.tmp` rename isn't safe under
// concurrent invocations).
type CfgStore struct {
	wmu sync.Mutex // serialises Mutate writers (and config.Save)
	p   atomic.Pointer[config.Config]
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

// Mutate atomically applies fn to the current config, persists it to disk
// via config.Save, and updates the in-memory pointer. Concurrent Mutate
// calls are serialised, so a read-modify-write (e.g. flip a single field
// inside fn) cannot lose updates against a parallel settings.set. fn must
// not call back into CfgStore — it would deadlock.
//
// If fn returns a non-nil error, Mutate aborts before Save and the
// in-memory pointer swap. This matters for validation that happens
// inside fn (e.g. json.Unmarshal of a renderer patch, which leaves the
// destination *partially* populated on a malformed payload — without
// this guard, a half-decoded config would be persisted to disk and
// served to all subsequent readers).
func (cs *CfgStore) Mutate(fn func(*config.Config) error) (config.Config, error) {
	cs.wmu.Lock()
	defer cs.wmu.Unlock()
	var cfg config.Config
	if cur := cs.p.Load(); cur != nil {
		cfg = *cur
	} else {
		cfg = config.Default()
	}
	if err := fn(&cfg); err != nil {
		return cfg, err
	}
	if err := config.Save(cfg); err != nil {
		return cfg, err
	}
	cs.p.Store(&cfg)
	return cfg, nil
}

// Settings provides settings get/set operations backed by CfgStore + config.Save.
type Settings struct {
	Store *CfgStore
}

// Get returns the current configuration as loaded by config.Load().
func (s *Settings) Get() config.Config {
	return s.Store.Get()
}

// Mutate is a thin pass-through to CfgStore.Mutate so the RPC layer can
// apply patches atomically without exposing CfgStore directly. If fn
// returns an error it is forwarded verbatim (so callers can distinguish
// validation failures from persistence failures).
func (s *Settings) Mutate(fn func(*config.Config) error) error {
	_, err := s.Store.Mutate(fn)
	return err
}
