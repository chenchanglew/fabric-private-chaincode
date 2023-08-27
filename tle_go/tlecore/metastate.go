package tlecore

import (
	"sync"

	"github.com/pkg/errors"
)

// metastate interface definition
type metastate interface {
	GetMeta(namespace string, key string) ([]byte, error)
	PutMeta(namespace string, key string, metadata []byte) error
}

// Tlestate struct implementing the metastate interface
type Tlestate struct {
	data  map[string]map[string][]byte
	mutex sync.Mutex
}

// GetMeta retrieves the metadata for the given namespace and key
func (t *Tlestate) GetMeta(namespace string, key string) ([]byte, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	metaMap, ok := t.data[namespace]
	if !ok {
		return nil, errors.Errorf("namespace not found: %s", namespace)
	}
	meta := metaMap[key]
	return meta, nil
}

// PutMeta stores the metadata for the given namespace and key
func (t *Tlestate) PutMeta(namespace string, key string, metadata []byte) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.data == nil {
		t.data = make(map[string]map[string][]byte)
	}

	if t.data[namespace] == nil {
		t.data[namespace] = make(map[string][]byte)
	}

	t.data[namespace][key] = metadata

	return nil
}
