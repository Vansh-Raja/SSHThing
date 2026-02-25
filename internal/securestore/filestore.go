package securestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type fileStore struct {
	mu   sync.Mutex
	path string
}

type fileStoreData struct {
	Entries map[string]string `json:"entries"`
}

var (
	globalFileStore     *fileStore
	globalFileStoreOnce sync.Once
)

func getFileStore() (*fileStore, error) {
	var initErr error
	globalFileStoreOnce.Do(func() {
		dir, err := os.UserConfigDir()
		if err != nil {
			initErr = err
			return
		}
		storePath := filepath.Join(dir, "sshthing", "keystore.json")
		storeDir := filepath.Dir(storePath)
		if err := os.MkdirAll(storeDir, 0700); err != nil {
			initErr = err
			return
		}
		globalFileStore = &fileStore{path: storePath}
	})
	if initErr != nil {
		return nil, initErr
	}
	if globalFileStore == nil {
		return nil, os.ErrNotExist
	}
	return globalFileStore, nil
}

func (fs *fileStore) load() (fileStoreData, error) {
	var data fileStoreData
	data.Entries = make(map[string]string)

	raw, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, err
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return fileStoreData{Entries: make(map[string]string)}, nil
	}
	if data.Entries == nil {
		data.Entries = make(map[string]string)
	}
	return data, nil
}

func (fs *fileStore) save(data fileStoreData) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fs.path, raw, 0600)
}

func (fs *fileStore) Get(service, user string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.load()
	if err != nil {
		return "", err
	}
	key := service + ":" + user
	val, ok := data.Entries[key]
	if !ok {
		return "", errNotFound
	}
	return val, nil
}

func (fs *fileStore) Set(service, user, value string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.load()
	if err != nil {
		return err
	}
	key := service + ":" + user
	data.Entries[key] = value
	return fs.save(data)
}

func (fs *fileStore) Delete(service, user string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := fs.load()
	if err != nil {
		return err
	}
	key := service + ":" + user
	if _, ok := data.Entries[key]; !ok {
		return errNotFound
	}
	delete(data.Entries, key)
	return fs.save(data)
}
