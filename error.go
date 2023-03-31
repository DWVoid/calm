package calm

import (
	"runtime"
)

type Error interface {
	// TCode : Top error type code
	TCode() uint32
	// ECode : Top error error code
	ECode() uint32
	// Slices : raw []ErrorInfo for all nesting sites
	Slices() []ErrorInfo
	// Error : same as detail, for compatibility with golang built-in error handling
	error
}

type ErrorInfo interface {
	// TCode : error type code
	TCode() uint32
	// ECode : error error code
	ECode() uint32
	// Detail : message for internal systems, e.g. logging
	Detail() string
	// Clean : sanitized short message for passing to the user as error detail
	Clean() string
	// Meta : optional metadata object for internal data
	Meta() any
}

type errNode struct {
	info  ErrorInfo
	trace []uintptr
}

func (e *errNode) TCode() uint32       { return e.info.TCode() }
func (e *errNode) ECode() uint32       { return e.info.ECode() }
func (e *errNode) Slices() []ErrorInfo { return []ErrorInfo{e.info} }
func (e *errNode) Error() string       { return PrintDetails(e, true) }

type errChain struct {
	errNode
	next any
}

func (e *errChain) Error() string { return PrintDetails(e, true) }

func (e *errChain) Slices() []ErrorInfo {
	var c any = e
	var b []ErrorInfo
	for {
		switch n := c.(type) {
		case *errChain:
			b = append(b, n.info)
			c = n.next
		case *errNode:
			b = append(b, n.info)
			return b
		}
	}
}

type eInfoCode struct {
	fCode uint64
}

func NewInfoCode(err uint64) ErrorInfo { return &eInfoCode{fCode: err} }

func (e *eInfoCode) TCode() uint32  { return uint32(e.fCode >> 32) }
func (e *eInfoCode) ECode() uint32  { return uint32(e.fCode) }
func (e *eInfoCode) Meta() any      { return nil }
func (e *eInfoCode) Detail() string { return e.Clean() }

func (e *eInfoCode) Clean() string {
	if reg, ok := _Reg.Load(e.TCode()); ok {
		return reg.(IErrType).DefaultCleanMsg(e.ECode())
	}
	return ""
}

type eInfoErr struct {
	eInfoCode
	err error
}

func (e *eInfoErr) Detail() string { return e.err.Error() }
func (e *eInfoErr) Clean() string  { return e.err.Error() }
func (e *eInfoErr) Meta() any      { return e.err }

func NewInfoErr(err uint64, nested error) ErrorInfo {
	return &eInfoErr{eInfoCode: eInfoCode{fCode: err}, err: nested}
}

type eInfoClean struct {
	eInfoCode
	msg string
}

func (e *eInfoClean) Detail() string { return e.msg }
func (e *eInfoClean) Clean() string  { return e.msg }

func NewInfoClean(err uint64, msg string) ErrorInfo {
	return &eInfoClean{eInfoCode: eInfoCode{fCode: err}, msg: msg}
}

func ErrNested(err uint64, nested error) Error {
	switch typed := nested.(type) {
	case Error:
		// special rule: if the fCode did not change, do nothing
		if (typed.TCode() == uint32(err>>32)) && (typed.ECode() == uint32(err)) {
			return typed
		}
		return ErrNestByInfo(typed, NewInfoCode(err))
	default:
		return ErrByInfo(NewInfoErr(err, nested))
	}
}

func ErrClean(err uint64, message string) Error {
	return ErrByInfo(NewInfoClean(err, message))
}

func _WithTrace(o bool) (result []uintptr) {
	if o {
		result = make([]uintptr, 2048)
		n := runtime.Callers(0, result)
		result = append([]uintptr(nil), result[:n]...)
	}
	return
}

// ErrNestByInfo : if info did not change since last frame, do nothing. otherwise, nest the given error with info
func ErrNestByInfo(nested Error, info ErrorInfo) Error {
	var errInfo ErrorInfo
	var errTrace []uintptr
	switch t := nested.(type) {
	case *errNode:
		errInfo = t.info
		errTrace = t.trace
	case *errChain:
		errInfo = t.info
		errTrace = t.trace
	}
	if errInfo == info {
		return nested
	}
	err := &errChain{errNode: errNode{info: info}, next: nested}
	if reg, ok := _Reg.Load(err.TCode()); ok {
		err.trace = _WithTrace(reg.(IErrType).OnErrNest(err) || (errTrace != nil))
	} else {
		err.trace = _WithTrace(errTrace != nil)
	}
	return err
}

func ErrByInfo(info ErrorInfo) Error {
	err := &errNode{info: info}
	if reg, ok := _Reg.Load(err.TCode()); ok {
		err.trace = _WithTrace(reg.(IErrType).OnErrRoot(err))
	}
	return err
}

func Unreachable[T any]() (ret T) {
	return
}

func Throw(err error) {
	panic(err)
}

func ThrowNested(err uint64, nested error) {
	Throw(ErrNested(err, nested))
}

func ThrowTagged(err uint64, message string) {
	Throw(ErrClean(err, message))
}
