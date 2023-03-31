package calm

import (
	"sync"
	"sync/atomic"
)

type Defer struct {
	err  Error
	done uint32
	m    sync.Mutex
}

type DeferT[T any] struct {
	val T
	Defer
}

func ValDefer() Defer {
	return Defer{err: nil, done: 1, m: sync.Mutex{}}
}

func ErrDefer(err error) Defer {
	return Defer{err: _AnyToError(err), done: 1, m: sync.Mutex{}}
}

func ErrDeferT[T any](err error) DeferT[T] {
	return DeferT[T]{Defer: ErrDefer(err)}
}

func ValDeferT[T any](v T) DeferT[T] {
	return DeferT[T]{val: v, Defer: ValDefer()}
}

func (c *Defer) realWait() {
	c.m.Lock()
	c.m.Unlock()
}

func (c *Defer) Wait() {
	if atomic.LoadUint32(&c.done) == 0 {
		// Outlined slow-path to allow inlining of the fast-path.
		c.realWait()
	}
}

func (c *DeferT[T]) Wait() {
	c.Defer.Wait()
}

func (c *Defer) Unwrap(onError func(Error)) {
	c.Wait()
	if c.err != nil {
		onError(c.err)
	}
}

func (c *Defer) Fold(onError func(Error)) {
	c.Wait()
	c.Unwrap(onError)
}

func (c *Defer) Get() {
	c.Wait()
	c.Unwrap(func(err Error) { Throw(err) })
}

func (c *Defer) Dump() Error {
	c.Wait()
	return c.err
}

func (c *DeferT[T]) Unwrap(onError func(Error)) T {
	c.Defer.Unwrap(onError)
	return c.val
}

func (c *DeferT[T]) Fold(onError func(Error) T) T {
	c.Wait()
	if c.Defer.err != nil {
		return onError(c.Defer.err)
	}
	return c.val
}

func (c *DeferT[T]) Get() T {
	c.Defer.Get()
	return c.val
}

func (c *DeferT[T]) Dump() (T, Error) {
	c.Wait()
	return c.val, c.Defer.err
}

func RunAsync(exec func()) (ret Defer) {
	ret = UnsafeMakeDefer()
	go ret.UnsafeApplyDefer(exec)
	return
}

func RunAsyncT[T any](exec func() T) (ret DeferT[T]) {
	ret = UnsafeMakeDeferT[T]()
	go ret.UnsafeApplyDefer(exec)
	return
}

func UnsafeMakeDefer() (ret Defer) {
	ret = Defer{err: nil, done: 0, m: sync.Mutex{}}
	ret.m.Lock()
	return
}

func (c *Defer) UnsafeApplyDefer(exec func()) {
	defer func() {
		o := recover()
		if o != nil {
			c.err = _AnyToError(o)
		}
		atomic.StoreUint32(&c.done, 1)
		c.m.Unlock()
	}()
	exec()
}

func UnsafeMakeDeferT[T any]() (ret DeferT[T]) {
	ret = DeferT[T]{Defer: Defer{err: nil, done: 0, m: sync.Mutex{}}}
	ret.Defer.m.Lock()
	return
}

func (c *DeferT[T]) UnsafeApplyDefer(exec func() T) {
	defer func() {
		o := recover()
		if o != nil {
			c.Defer.err = _AnyToError(o)
		}
		atomic.StoreUint32(&c.Defer.done, 1)
		c.Defer.m.Unlock()
	}()
	c.val = exec()
}
