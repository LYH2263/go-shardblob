# go-shardblob

内容寻址分片对象仓：把字节流切成定长或 CDC 分片，按内容哈希去重落盘，维护清单、引用计数与 GC。无前端。

## 不变量

1. 相同内容分片只存一份。
2. 读回拼接后的摘要等于写入摘要。
3. GC 不得删除 `refcount > 0` 的分片。
4. 半写清单不进入索引，对外不可见。

## 使用

```go
s, err := shardblob.Open(dir, shardblob.WithChunkSize(64<<10))
id, err := s.Put(reader)
rc, err := s.Get(id)
s.Verify(id)
s.Delete(id)
s.GC()
```

`OpenMemory` 提供纯内存后端。分片大小可配，尾块允许短于片长。

## 测试

```bash
go test ./... -count=1
```
