package verify

import (
	"fmt"
)

// Kind 故障类别。
type Kind string

const (
	KindChunkHash  Kind = "chunk_hash"
	KindChunkMiss  Kind = "chunk_missing"
	KindChunkSize  Kind = "chunk_size"
	KindObjectHash Kind = "object_hash"
	KindManifest   Kind = "manifest"
	KindAlgo       Kind = "algo"
)

// Fault 单条校验失败。
type Fault struct {
	Kind   Kind
	Chunk  int
	Detail string
}

func (f Fault) String() string {
	if f.Chunk >= 0 {
		return fmt.Sprintf("%s chunk=%d %s", f.Kind, f.Chunk, f.Detail)
	}
	return fmt.Sprintf("%s %s", f.Kind, f.Detail)
}

// Report 对象校验报告。
type Report struct {
	ObjectHex string
	Chunks    int
	Bytes     uint64
	Faults    []Fault
}

func (r Report) OK() bool { return len(r.Faults) == 0 }

func (r Report) Error() string {
	if r.OK() {
		return ""
	}
	return fmt.Sprintf("verify: object %s faults=%d first=%s", r.ObjectHex, len(r.Faults), r.Faults[0].String())
}

func (r *Report) add(k Kind, chunk int, detail string) {
	r.Faults = append(r.Faults, Fault{Kind: k, Chunk: chunk, Detail: detail})
}
