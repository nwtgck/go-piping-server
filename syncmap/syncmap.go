package syncmap

import "sync"

type SyncMap[K any, V any] struct {
	inner sync.Map
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	actualAny, loaded := m.inner.LoadOrStore(key, value)
	actual = actualAny.(V)
	return
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.inner.Delete(key)
}
