package blobstore

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/layout"
)

// Disk 把分片落到 root/chunks/ab/cd/<hex>。写入走临时文件再改名。
type Disk struct {
	root   string
	mu     sync.RWMutex
	nbytes atomic.Int64
	nfile  atomic.Int64
	closed atomic.Bool
}

func OpenDisk(root string) (*Disk, error) {
	if err := layout.EnsureRoot(root); err != nil {
		return nil, err
	}
	d := &Disk{root: root}
	if err := d.recount(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Disk) recount() error {
	var files, bytes int64
	err := filepath.Walk(filepath.Join(d.root, layout.DirChunks), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		base := info.Name()
		if strings.HasPrefix(base, ".") || strings.HasSuffix(base, layout.TmpSuffix) {
			return nil
		}
		if len(base) != 64 {
			return nil
		}
		files++
		bytes += info.Size()
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	d.nfile.Store(files)
	d.nbytes.Store(bytes)
	return nil
}

func (d *Disk) guard() error {
	if d.closed.Load() {
		return ErrClosed
	}
	return nil
}

func (d *Disk) Put(id hashx.ID, data []byte) error {
	if err := d.guard(); err != nil {
		return err
	}
	path := layout.ChunkPath(d.root, id)
	if layout.Exists(path) {
		return nil
	}
	if err := layout.WriteAtomicExclusive(path, data); err != nil {
		return err
	}
	d.nfile.Add(1)
	d.nbytes.Add(int64(len(data)))
	return nil
}

func (d *Disk) Get(id hashx.ID) ([]byte, error) {
	if err := d.guard(); err != nil {
		return nil, err
	}
	path := layout.ChunkPath(d.root, id)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (d *Disk) Open(id hashx.ID) (io.ReadCloser, error) {
	if err := d.guard(); err != nil {
		return nil, err
	}
	path := layout.ChunkPath(d.root, id)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (d *Disk) Has(id hashx.ID) (bool, error) {
	if err := d.guard(); err != nil {
		return false, err
	}
	return layout.Exists(layout.ChunkPath(d.root, id)), nil
}

func (d *Disk) Delete(id hashx.ID) error {
	if err := d.guard(); err != nil {
		return err
	}
	path := layout.ChunkPath(d.root, id)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	sz := info.Size()
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	d.nfile.Add(-1)
	d.nbytes.Add(-sz)
	return nil
}

func (d *Disk) List(fn func(hashx.ID) error) error {
	if err := d.guard(); err != nil {
		return err
	}
	root := filepath.Join(d.root, layout.DirChunks)
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		base := info.Name()
		if strings.HasPrefix(base, ".") || strings.HasSuffix(base, layout.TmpSuffix) {
			return nil
		}
		if len(base) != 64 {
			return nil
		}
		raw, decErr := hex.DecodeString(base)
		if decErr != nil {
			return fmt.Errorf("blobstore: bad chunk name %s: %w", base, decErr)
		}
		id, idErr := hashx.FromSlice(raw)
		if idErr != nil {
			return idErr
		}
		return fn(id)
	})
}

func (d *Disk) Bytes() int64 { return d.nbytes.Load() }

func (d *Disk) Count() int { return int(d.nfile.Load()) }

func (d *Disk) Close() error {
	d.closed.Store(true)
	return nil
}

func (d *Disk) Root() string { return d.root }
