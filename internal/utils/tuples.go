package utils

import "sync"

// Generic Pair and Triple helpers for small tuple-like structs.
// Pair uses fields I and J to be compatible with existing Position usages.

type Pair[A any, B any] struct {
	I A
	J B
}

func NewPair[A any, B any](i A, j B) Pair[A, B] {
	return Pair[A, B]{I: i, J: j}
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
