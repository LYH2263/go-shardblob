package shardblob

import (
	"os"

	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/layout"
	"github.com/LYH2263/go-shardblob/internal/manifest"
)

func readManifestFile(root string, id hashx.ID) (*manifest.Manifest, error) {
	path := layout.ManifestPath(root, id)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	mf, err := manifest.Decode(raw)
	if err != nil {
		return nil, err
	}
	return mf, nil
}

func writeManifestFile(root string, id hashx.ID, raw []byte) error {
	return layout.WriteAtomic(layout.ManifestPath(root, id), raw)
}
