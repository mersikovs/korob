// Package pool is an example of a generic pool built on top of sync.Pool.
//
// sync.Pool stores any and cannot call Reset on its own.
// The Pool[T] wrapper does two things:
//   - it stores the type T statically (no need for .(*Foo) in each call);
//   - it ensures that Put calls Reset before returning to the pool.
package pool

import "sync"

// Resettable — types that can be stored in the Pool.
type Resettable interface {
	Reset()
}

// Pool — a generic wrapper over sync.Pool.
type Pool[T Resettable] struct {
	inner sync.Pool
}

// New creates a pool. newFn is called when the pool is empty.
// newFn must not be nil and must not return nil T.
func New[T Resettable](newFn func() T) *Pool[T] {
	return &Pool[T]{
		inner: sync.Pool{
			New: func() any { return newFn() },
		},
	}
}

// Get returns an object from the pool or creates a new one via newFn.
func (p *Pool[T]) Get() T {
	return p.inner.Get().(T)
}

// Put resets v and returns it to the pool.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.inner.Put(v)
}
