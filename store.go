package shardblob

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/chunk"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/index"
	"github.com/LYH2263/go-shardblob/internal/iofence"
	"github.com/LYH2263/go-shardblob/internal/layout"
	"github.com/LYH2263/go-shardblob/internal/refcount"
)

// Store 内容寻址分片对象仓。
type Store struct {
	dir      string
	opts     options
	algo     hashx.Algo
	blobs    blobstore.Backend
	refs     *refcount.Table
	idx      *index.Index
	fence    *iofence.Fence
	journal  *refcount.Journal
	mem      *memMan
	persistM sync.Mutex
	closed   bool
}

// Open 打开（或创建）磁盘仓。root 必须非空。
func Open(root string, opts ...Option) (*Store, error) {
	if root == "" {
		return nil, fmt.Errorf("%w: empty root", ErrInvalid)
	}
	o, err := defaultOptions().apply(opts)
	if err != nil {
		return nil, err
	}
	if err := layout.EnsureRoot(root); err != nil {
		return nil, err
	}
	want := layout.Config{
		Algo:      o.algoName,
		ChunkSize: o.policy.Avg,
		CDC:       o.policy.Mode == chunk.ModeCDC,
	}
	have, ok, err := layout.LoadConfig(root)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := layout.Compatible(have, want); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrConfig, err)
		}
	} else {
		if err := layout.SaveConfig(root, want); err != nil {
			return nil, err
		}
	}
	algo, err := hashx.Lookup(o.algoName)
	if err != nil {
		return nil, err
	}
	disk, err := blobstore.OpenDisk(root)
	if err != nil {
		return nil, err
	}
	idx, err := index.Load(layout.IndexPath(root))
	if err != nil {
		_ = disk.Close()
		return nil, err
	}
	refs, err := refcount.Load(layout.RefsPath(root))
	if err != nil {
		_ = disk.Close()
		return nil, err
	}
	j, err := refcount.OpenJournal(layout.JournalPath(root))
	if err != nil {
		_ = disk.Close()
		return nil, err
	}
	if err := j.Replay(refs); err != nil {
		_ = j.Close()
		_ = disk.Close()
		return nil, err
	}
	refs.AttachJournal(j)
	s := &Store{
		dir:     root,
		opts:    o,
		algo:    algo,
		blobs:   disk,
		refs:    refs,
		idx:     idx,
		fence:   iofence.New(),
		journal: j,
	}
	return s, nil
}

// OpenMemory 纯内存仓，不落盘。
func OpenMemory(opts ...Option) (*Store, error) {
	o, err := defaultOptions().apply(append([]Option{withMemory()}, opts...))
	if err != nil {
		return nil, err
	}
	algo, err := hashx.Lookup(o.algoName)
	if err != nil {
		return nil, err
	}
	return &Store{
		opts:  o,
		algo:  algo,
		blobs: blobstore.NewMemory(),
		refs:  refcount.NewTable(),
		idx:   index.New(),
		fence: iofence.New(),
		mem:   newMemMan(),
	}, nil
}

func (s *Store) beginIO() (func(), error) {
	if s == nil {
		return nil, ErrClosed
	}
	done := s.fence.BeginShared()
	if s.closed {
		done()
		return nil, ErrClosed
	}
	return done, nil
}

func (s *Store) ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// abortNewChunks 删除本轮新写入的分片，用于 Put 取消或出错后的回滚。
// 调用方应只传入本次 Put 新建（此前不存在）的分片 ID，以免误删被其他对象引用的分片。
// 删除为尽力而为：即便某个分片删除失败也继续处理其余分片，并返回首个遇到的错误。
func (s *Store) abortNewChunks(ids []hashx.ID) error {
	var first error
	for _, id := range ids {
		if err := s.blobs.Delete(id); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *Store) writeManifest(id hashx.ID, raw []byte) error {
	if s.opts.memory {
		return s.putMemManifest(id, raw)
	}
	return writeManifestFile(s.dir, id, raw)
}

func (s *Store) removeManifest(id hashx.ID) error {
	if s.opts.memory {
		s.delMemManifest(id)
		return nil
	}
	return layout.RemoveAllIfExist(layout.ManifestPath(s.dir, id))
}

func (s *Store) persist() error {
	s.persistM.Lock()
	defer s.persistM.Unlock()
	if s.opts.memory || s.dir == "" {
		return nil
	}
	if err := index.Save(layout.IndexPath(s.dir), s.idx); err != nil {
		return err
	}
	if err := refcount.Save(layout.RefsPath(s.dir), s.refs); err != nil {
		return err
	}
	if s.journal != nil {
		if err := s.journal.Truncate(); err != nil {
			return err
		}
	}
	s.refs.MarkClean()
	return nil
}

// Close 刷盘并关闭后端。
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	done := s.fence.BeginExclusive()
	defer done()
	if s.closed {
		return nil
	}
	s.closed = true
	err := s.persist()
	if s.journal != nil {
		if cErr := s.journal.Close(); err == nil {
			err = cErr
		}
	}
	if s.blobs != nil {
		if cErr := s.blobs.Close(); err == nil {
			err = cErr
		}
	}
	return err
}

// Dir 返回磁盘根；内存仓为空串。
func (s *Store) Dir() string { return s.dir }

// ConfigJSON 当前配置（调试）。
func (s *Store) ConfigJSON() string {
	c := layout.Config{
		Algo:      s.opts.algoName,
		ChunkSize: s.opts.policy.Avg,
		CDC:       s.opts.policy.Mode == chunk.ModeCDC,
	}
	b, _ := json.Marshal(c)
	return string(b)
}
