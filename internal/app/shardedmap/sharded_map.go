package shardedmap

import (
	"sync"
)

type Shard[T any] struct {
	data map[any]T
	mu   sync.RWMutex
}

func (s *Shard[T]) Get(key any) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.data[key]
	return value, exists
}

func (s *Shard[T]) Set(key any, value T) {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
}

func (s *Shard[T]) Remove(key any) {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
}

type ShardedMap[T any] struct {
	shards map[any]*Shard[T]
	mu     sync.RWMutex
}

func New[T any]() *ShardedMap[T] {
	return &ShardedMap[T]{shards: make(map[any]*Shard[T])}
}

func (s *ShardedMap[T]) GetShard(key any) (*Shard[T], bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	shard, exists := s.shards[key]
	return shard, exists
}

func (s *ShardedMap[T]) NewShard(key any) *Shard[T] {
	s.mu.Lock()
	defer s.mu.Unlock()

	if shard, exists := s.shards[key]; exists {
		return shard
	}

	shard := &Shard[T]{data: make(map[any]T)}
	s.shards[key] = shard
	return shard
}
