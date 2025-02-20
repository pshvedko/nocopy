package internal

import "sync"

type Map[K comparable, V any] struct {
	w sync.WaitGroup
	m sync.Map
}

func (m *Map[K, V]) Store(key K, value V) (ok bool) {
	m.w.Add(1)
	_, ok = m.m.LoadOrStore(key, value)
	if ok {
		m.w.Done()
	}
	return !ok
}

func (m *Map[K, V]) Delete(key K) (value V, ok bool) {
	v, ok := m.m.LoadAndDelete(key)
	if ok {
		value = v.(V)
		m.w.Done()
	}
	return
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (m *Map[K, V]) Wait() {
	m.w.Wait()
}
