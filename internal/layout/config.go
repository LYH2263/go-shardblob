package layout

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/LYH2263/go-shardblob/internal/chunk"
)

// Config 仓级切分与摘要配置，Open 时写入，重开必须一致。
type Config struct {
	Algo      string `json:"algo"`
	ChunkSize int    `json:"chunk_size"`
	CDC       bool   `json:"cdc"`
}

func (c Config) Policy() chunk.Policy {
	mode := chunk.ModeFixed
	if c.CDC {
		mode = chunk.ModeCDC
	}
	return chunk.Normalize(chunk.Policy{Mode: mode, Avg: c.ChunkSize})
}

// LoadConfig 读取 meta/config；不存在返回 ok=false。
func LoadConfig(root string) (Config, bool, error) {
	var c Config
	path := ConfigPath(root)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, false, nil
		}
		return c, false, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, false, fmt.Errorf("layout: config json: %w", err)
	}
	return c, true, nil
}

// SaveConfig 原子写入配置。
func SaveConfig(root string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return WriteAtomic(ConfigPath(root), append(b, '\n'))
}

// Compatible 报告已存配置与请求是否一致。
func Compatible(have, want Config) error {
	if have.Algo != want.Algo {
		return fmt.Errorf("layout: algo mismatch have=%s want=%s", have.Algo, want.Algo)
	}
	if have.ChunkSize != want.ChunkSize {
		return fmt.Errorf("layout: chunk size mismatch have=%d want=%d", have.ChunkSize, want.ChunkSize)
	}
	if have.CDC != want.CDC {
		return fmt.Errorf("layout: cdc mismatch have=%v want=%v", have.CDC, want.CDC)
	}
	return nil
}
