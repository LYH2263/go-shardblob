package shardblob

// StoreStats 返回对象数、分片数与去重占用。
func (s *Store) StoreStats() (Stats, error) {
	done, err := s.beginIO()
	if err != nil {
		return Stats{}, err
	}
	defer done()
	return Stats{
		Objects:     s.idx.Len(),
		Chunks:      s.blobs.Count(),
		Logical:     s.idx.TotalBytes(),
		UniqueBytes: s.blobs.Bytes(),
	}, nil
}

// List 列出全部对象元数据。
func (s *Store) List() ([]Info, error) {
	done, err := s.beginIO()
	if err != nil {
		return nil, err
	}
	defer done()
	recs := s.idx.List()
	out := make([]Info, 0, len(recs))
	for _, rec := range recs {
		out = append(out, Info{
			ID:       fromHX(rec.ID),
			Size:     rec.Size,
			Chunks:   int(rec.Chunks),
			UseCount: rec.UseCount,
		})
	}
	return out, nil
}

// ChunkRef 返回分片引用计数（测试/运维）。
func (s *Store) ChunkRef(hexID string) (uint64, error) {
	done, err := s.beginIO()
	if err != nil {
		return 0, err
	}
	defer done()
	id, err := ParseObjectID(hexID)
	if err != nil {
		return 0, err
	}
	return s.refs.Get(toHX(id)), nil
}
