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

func (lv *LockedValue) Incr() {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	lv.value += 1
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
