package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic 先写同目录临时文件再 Rename，避免半写可见。
func WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("layout: mkdir %s: %w", dir, err)
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("layout: create temp: %w", err)
	}
	tmp := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("layout: write temp: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("layout: sync temp: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("layout: close temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		// Windows 上目标已存在时 Rename 可能失败；内容寻址场景视为成功。
		if _, statErr := os.Stat(path); statErr == nil {
			cleanup = true
			return nil
		}
		return fmt.Errorf("layout: rename %s -> %s: %w", tmp, path, err)
	}
	cleanup = false
	return nil
}

// WriteAtomicExclusive 若目标已存在则跳过写入。
func WriteAtomicExclusive(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return WriteAtomic(path, data)
}

// ReadFile 读取整个文件。
func ReadFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// RemoveAllIfExist 删除路径（不存在不算错）。
func RemoveAllIfExist(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

// Exists 报告路径是否存在。
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
