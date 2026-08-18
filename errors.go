package shardblob

import "errors"

var (
	ErrClosed     = errors.New("shardblob: store closed")
	ErrNotFound   = errors.New("shardblob: object not found")
	ErrCorrupt    = errors.New("shardblob: object corrupt")
	ErrInvalid    = errors.New("shardblob: invalid argument")
	ErrConfig     = errors.New("shardblob: store config mismatch")
	ErrIncomplete = errors.New("shardblob: incomplete manifest")
	ErrVerify     = errors.New("shardblob: verify failed")
)

func closedOr(err error) error {
	if err == nil {
		return ErrClosed
	}
	return err
}
