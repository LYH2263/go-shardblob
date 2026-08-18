package shardblob

import (
	"fmt"

	"github.com/LYH2263/go-shardblob/internal/verify"
)

// Verify 重算对象全部分片与拼接摘要。
func (s *Store) Verify(id ObjectID) error {
	done, err := s.beginIO()
	if err != nil {
		return err
	}
	defer done()
	hid := toHX(id)
	if !s.idx.Has(hid) {
		return ErrNotFound
	}
	mf, err := s.loadManifest(hid)
	if err != nil {
		return err
	}
	rep, err := verify.Object(s.blobs, mf)
	if err != nil {
		return err
	}
	_ = rep
	return nil
}

// VerifyAll 校验索引中全部对象。
func (s *Store) VerifyAll() error {
	done, err := s.beginIO()
	if err != nil {
		return err
	}
	defer done()
	for _, rec := range s.idx.List() {
		mf, err := s.loadManifest(rec.ID)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrVerify, rec.ID.Hex(), err)
		}
		if _, err := verify.Object(s.blobs, mf); err != nil {
			return fmt.Errorf("%w: %v", ErrVerify, err)
		}
	}
	return nil
}
