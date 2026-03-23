package utils

import (
	"sync"
)

type LockedBool struct {
	mu    sync.Mutex
	value bool
}

func NewLockedBool(value bool) *LockedBool {
	return &LockedBool{value: value}
}

func (lb *LockedBool) Get() bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.value
}

func (lb *LockedBool) Set(value bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.value = value
}

func (lb *LockedBool) CompareAndSet(value bool) bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if lb.value == value {
		return false
	}
	lb.value = value
	return true
}

type LockedValue struct {
	mutex sync.Mutex
	value int
}

func NewLockedValue(value int) *LockedValue {
	return &LockedValue{value: value}
}

func (lv *LockedValue) Get() int {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	return lv.value
}

func (lv *LockedValue) Set(value int) {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	lv.value = value
}

func (lv *LockedValue) CompareAndIncrement(max_value int) bool {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	if lv.value >= max_value {
		return false
	}
	lv.value += 1
	return true
}

func (lv *LockedValue) GetAndIncrement() int {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	current_value := lv.value
	lv.value += 1
	return current_value
}

func (lv *LockedValue) Incr() {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	lv.value += 1
}

func (lv *LockedValue) Decr() {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	lv.value -= 1
}

type LockedPointer[T any] struct {
	mutex sync.Mutex
	value *T
}

func NewLockedPointer[T any](value *T) *LockedPointer[T] {
	return &LockedPointer[T]{value: value}
}

func (lp *LockedPointer[T]) Get() *T {
	lp.mutex.Lock()
	defer lp.mutex.Unlock()
	return lp.value
}

func (lp *LockedPointer[T]) Set(value *T) {
	lp.mutex.Lock()
	defer lp.mutex.Unlock()
	lp.value = value
}

type LockedMap[K comparable, V any] struct {
	mutex sync.Mutex
	value map[K]V
}

func NewLockedMap[K comparable, V any]() *LockedMap[K, V] {
	return &LockedMap[K, V]{value: make(map[K]V)}
}

func (lm *LockedMap[K, V]) Get(key K) (V, bool) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	value, ok := lm.value[key]
	return value, ok
}

func (lm *LockedMap[K, V]) Set(key K, value V) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	lm.value[key] = value
}

func (lm *LockedMap[K, V]) GetKeys() []K {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	keys := make([]K, 0, len(lm.value))
	for key := range lm.value {
		keys = append(keys, key)
	}
	return keys
}

func (lm *LockedMap[K, V]) Delete(key K) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	delete(lm.value, key)
}
