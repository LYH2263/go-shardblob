package shardblob

import (
	"sync"

	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/manifest"
)

type memMan struct {
	mu   sync.Mutex
	raws map[hashx.ID][]byte
}

func newMemMan() *memMan {
	return &memMan{raws: make(map[hashx.ID][]byte)}
}

func (s *Store) ensureMem() {
	if s.mem == nil {
		s.mem = newMemMan()
	}
}

func (s *Store) memManifest(id hashx.ID) (*manifest.Manifest, error) {
	s.ensureMem()
	s.mem.mu.Lock()
	raw, ok := s.mem.raws[id]
	s.mem.mu.Unlock()
	if !ok {
		return nil, ErrNotFound
	}
	return manifest.Decode(raw)
}

func (s *Store) putMemManifest(id hashx.ID, raw []byte) error {
	s.ensureMem()
	cp := make([]byte, len(raw))
	copy(cp, raw)
	s.mem.mu.Lock()
	s.mem.raws[id] = cp
	s.mem.mu.Unlock()
	return nil
}

func (s *Store) delMemManifest(id hashx.ID) {
	s.ensureMem()
	s.mem.mu.Lock()
	delete(s.mem.raws, id)
	s.mem.mu.Unlock()
}

func (s *Store) loadManifest(id hashx.ID) (*manifest.Manifest, error) {
	if s.opts.memory {
		return s.memManifest(id)
	}
	return readManifestFile(s.dir, id)
}
