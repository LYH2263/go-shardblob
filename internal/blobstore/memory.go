package blobstore

import (
	"bytes"
	"io"
	"sync"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

type memblob struct {
	data []byte
}

// Memory 内存后端，测试与无盘模式使用。
type Memory struct {
	mu     sync.RWMutex
	blobs  map[hashx.ID]memblob
	nbytes int64
	closed bool
}

func NewMemory() *Memory {
	return &Memory{blobs: make(map[hashx.ID]memblob)}
}

func (m *Memory) guard() error {
	if m.closed {
		return ErrClosed
	}
	return nil
}

func (m *Memory) Put(id hashx.ID, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.guard(); err != nil {
		return err
	}
	if _, ok := m.blobs[id]; ok {
		return nil
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.blobs[id] = memblob{data: cp}
	m.nbytes += int64(len(cp))
	return nil
}

func (m *Memory) Get(id hashx.ID) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.guard(); err != nil {
		return nil, err
	}
	b, ok := m.blobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(b.data))
	copy(cp, b.data)
	return cp, nil
}

func (m *Memory) Open(id hashx.ID) (io.ReadCloser, error) {
	data, err := m.Get(id)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *Memory) Has(id hashx.ID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.guard(); err != nil {
		return false, err
	}
	_, ok := m.blobs[id]
	return ok, nil
}

func (m *Memory) Delete(id hashx.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.guard(); err != nil {
		return err
	}
	b, ok := m.blobs[id]
	if !ok {
		return nil
	}
	m.nbytes -= int64(len(b.data))
	delete(m.blobs, id)
	return nil
}

func (m *Memory) List(fn func(hashx.ID) error) error {
	m.mu.RLock()
	ids := make([]hashx.ID, 0, len(m.blobs))
	for id := range m.blobs {
		ids = append(ids, id)
	}
	closed := m.closed
	m.mu.RUnlock()
	if closed {
		return ErrClosed
	}
	for _, id := range ids {
		if err := fn(id); err != nil {
			return err
		}
	}
	return nil
}

func (m *Memory) Bytes() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nbytes
}

func (m *Memory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.blobs)
}

func (m *Memory) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
