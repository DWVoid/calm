package calm

import "fmt"

type Result struct {
	err Error
}

type ResultT[T any] struct {
	val T
	Result
}

func Wrap(err error) Result {
	return Result{err: _AnyToError(err)}
}

func WrapT[T any](v T, err error) ResultT[T] {
	return ResultT[T]{val: v, Result: Wrap(err)}
}

func ValResult() Result {
	return Wrap(nil)
}

func ErrResult(err error) Result {
	return Wrap(err)
}

func ErrResultT[T any](err error) ResultT[T] {
	return ResultT[T]{Result: Wrap(err)}
}

func ValResultT[T any](v T) ResultT[T] {
	return ResultT[T]{val: v, Result: Wrap(nil)}
}

func (c Result) Unwrap(onError func(Error)) {
	if c.err != nil {
		onError(c.err)
	}
}

func (c Result) Fold(onError func(Error)) {
	c.Unwrap(onError)
}

func (c Result) Get() {
	c.Unwrap(func(err Error) { Throw(err) })
}

func (c Result) Dump() Error {
	return c.err
}

func (c ResultT[T]) Unwrap(onError func(Error)) T {
	c.Result.Unwrap(onError)
	return c.val
}

func (c ResultT[T]) Fold(onError func(Error) T) T {
	if c.Result.err != nil {
		return onError(c.Result.err)
	}
	return c.val
}

func (c ResultT[T]) Get() T {
	c.Result.Get()
	return c.val
}

func (c ResultT[T]) Dump() (T, Error) {
	return c.val, c.Result.Dump()
}

func _AnyToError(o any) Error {
	if o == nil {
		return nil
	}
	switch err := o.(type) {
	case Error:
		return err
	case error:
		return NestedError(EInternal, err)
	default:
		return TaggedError(EInternal, "panic:"+fmt.Sprint(o))
	}
}

func Run(exec func()) (ret Result) {
	defer func() {
		o := recover()
		if o != nil {
			ret = ErrResult(_AnyToError(o))
		}
	}()
	exec()
	return ValResult()
}

func RunT[T any](exec func() T) (ret ResultT[T]) {
	defer func() {
		o := recover()
		if o != nil {
			ret = ErrResultT[T](_AnyToError(o))
		}
	}()
	return ValResultT(exec())
}
