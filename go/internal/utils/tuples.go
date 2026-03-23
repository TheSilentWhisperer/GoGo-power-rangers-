package utils

import (
	"sync"

	"github.com/gammazero/deque"
)

// Generic Pair and Triple helpers for small tuple-like structs.
// Pair uses fields I and J to be compatible with existing Position usages.

type Pair[A any, B any] struct {
	First  A
	Second B
}

func NewPair[A any, B any](a A, b B) Pair[A, B] {
	return Pair[A, B]{First: a, Second: b}
}

type Triple[A any, B any, C any] struct {
	First  A
	Second B
	Third  C
}

func NewTriple[A any, B any, C any](a A, b B, c C) Triple[A, B, C] {
	return Triple[A, B, C]{First: a, Second: b, Third: c}
}

type LockedPair[A any, B any] struct {
	Pair  Pair[A, B]
	Mutex sync.Mutex
}

func NewLockedPair[A any, B any](i A, j B) *LockedPair[A, B] {
	return &LockedPair[A, B]{
		Pair:  NewPair(i, j),
		Mutex: sync.Mutex{},
	}
}

type LockedTriple[A any, B any, C any] struct {
	Triple Triple[A, B, C]
	Mutex  sync.Mutex
}

func NewLockedTriple[A any, B any, C any](a A, b B, c C) *LockedTriple[A, B, C] {
	return &LockedTriple[A, B, C]{
		Triple: NewTriple(a, b, c),
		Mutex:  sync.Mutex{},
	}
}

type LockedQueue[T any] struct {
	Mutex  sync.Mutex
	MaxCap int
	Queue  *deque.Deque[T]
}

func NewLockedQueue[T any](max_cap int) *LockedQueue[T] {
	var lq *LockedQueue[T] = &LockedQueue[T]{
		MaxCap: max_cap,
		Queue:  new(deque.Deque[T]),
		Mutex:  sync.Mutex{},
	}
	lq.Queue.SetBaseCap(max_cap)
	lq.Queue.Grow(max_cap)
	return lq
}

func (lq *LockedQueue[T]) Enqueue(item T) {
	lq.Mutex.Lock()
	defer lq.Mutex.Unlock()
	if lq.Queue.Len() == lq.MaxCap {
		println("Warning: LockedQueue is full, overwriting oldest item")
		lq.Queue.PopFront()
	}
	lq.Queue.PushBack(item)
}

func (lq *LockedQueue[T]) Dequeue() (T, bool) {
	lq.Mutex.Lock()
	defer lq.Mutex.Unlock()
	if lq.Queue.Len() == 0 {
		var zero T
		return zero, false
	}
	return lq.Queue.PopFront(), true
}

func (lq *LockedQueue[T]) Len() int {
	lq.Mutex.Lock()
	defer lq.Mutex.Unlock()
	return lq.Queue.Len()
}
