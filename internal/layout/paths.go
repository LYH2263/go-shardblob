package layout

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

const (
	DirChunks    = "chunks"
	DirManifests = "manifests"
	DirMeta      = "meta"
	FileConfig   = "config"
	FileIndex    = "index.bin"
	FileRefs     = "refs.bin"
	FileJournal  = "refs.journal"
	TmpSuffix    = ".tmp"
	MfsSuffix    = ".mfs"
)

// EnsureRoot 创建仓目录骨架。
func EnsureRoot(root string) error {
	if root == "" {
		return fmt.Errorf("layout: empty root")
	}
	for _, d := range []string{
		root,
		filepath.Join(root, DirChunks),
		filepath.Join(root, DirManifests),
		filepath.Join(root, DirMeta),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("layout: mkdir %s: %w", d, err)
		}
	}
	return nil
}

// ChunkRel 返回分片相对路径：chunks/ab/cd/<hex>。
func ChunkRel(id hashx.ID) string {
	h := hex.EncodeToString(id[:])
	return filepath.Join(DirChunks, h[0:2], h[2:4], h)
}

// ChunkPath 返回分片绝对路径。
func ChunkPath(root string, id hashx.ID) string {
	return filepath.Join(root, ChunkRel(id))
}

// ChunkDir 返回分片所在目录。
func ChunkDir(root string, id hashx.ID) string {
	h := hex.EncodeToString(id[:])
	return filepath.Join(root, DirChunks, h[0:2], h[2:4])
}

// ManifestPath 对象清单路径。
func ManifestPath(root string, id hashx.ID) string {
	h := hex.EncodeToString(id[:])
	return filepath.Join(root, DirManifests, h+MfsSuffix)
}

// ManifestTmp 清单半写路径（不得对外可见）。
func ManifestTmp(root string, id hashx.ID) string {
	return ManifestPath(root, id) + TmpSuffix
}

// ConfigPath 仓配置。
func ConfigPath(root string) string {
	return filepath.Join(root, DirMeta, FileConfig)
}

// IndexPath 对象索引快照。
func IndexPath(root string) string {
	return filepath.Join(root, DirMeta, FileIndex)
}

// RefsPath 引用计数快照。
func RefsPath(root string) string {
	return filepath.Join(root, DirMeta, FileRefs)
}

// JournalPath 引用计数日志。
func JournalPath(root string) string {
	return filepath.Join(root, DirMeta, FileJournal)
}

// FanoutOK 检查 hex 分片路径深度是否为 2 级扇出。
func FanoutOK(rel string) bool {
	parts := splitPath(rel)
	return len(parts) >= 4 && parts[0] == DirChunks
}

func splitPath(rel string) []string {
	var out []string
	for rel != "" && rel != "." {
		base := filepath.Base(rel)
		out = append([]string{base}, out...)
		next := filepath.Dir(rel)
		if next == rel {
			break
		}
		rel = next
	}
	return out
}

// HexID 把 ID 编成 64 hex。
func HexID(id hashx.ID) string {
	return hex.EncodeToString(id[:])
}
