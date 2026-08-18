package shardblob

import (
	"fmt"

	"github.com/LYH2263/go-shardblob/internal/chunk"
	"github.com/LYH2263/go-shardblob/internal/hashx"
)

type options struct {
	algoName string
	policy   chunk.Policy
	autoGC   bool
	memory   bool
}

func defaultOptions() options {
	return options{
		algoName: hashx.NameSHA256,
		policy:   chunk.DefaultPolicy(),
	}
}

func (o options) apply(opts []Option) (options, error) {
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	o.policy = chunk.Normalize(o.policy)
	if err := o.policy.Valid(); err != nil {
		return o, err
	}
	if _, err := hashx.Lookup(o.algoName); err != nil {
		return o, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return o, nil
}

// Option 打开仓时的选项。
type Option func(*options)

// WithAlgo 选择 sha256 或 sha512_256。
func WithAlgo(name string) Option {
	return func(o *options) { o.algoName = name }
}

// WithChunkSize 设置平均/定长分片大小（字节）。
func WithChunkSize(n int) Option {
	return func(o *options) { o.policy.Avg = n }
}

// WithCDC 启用内容定义切分。
func WithCDC(on bool) Option {
	return func(o *options) {
		if on {
			o.policy.Mode = chunk.ModeCDC
		} else {
			o.policy.Mode = chunk.ModeFixed
		}
	}
}

// WithAutoGC 在 Delete 后立即回收不可达分片。
func WithAutoGC(on bool) Option {
	return func(o *options) { o.autoGC = on }
}

func withMemory() Option {
	return func(o *options) { o.memory = true }
}
