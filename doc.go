// Package shardblob 是内容寻址的分片对象仓：把字节流切成定长或 CDC 分片，
// 按内容哈希去重存储，维护清单、引用计数与 GC。无前端。
package shardblob
