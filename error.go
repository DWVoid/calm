package calm

import (
	"fmt"
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

type sErrNode struct {
	info  ErrorInfo
	trace []uintptr
}

func (e *sErrNode) TCode() uint32       { return e.info.TCode() }
func (e *sErrNode) ECode() uint32       { return e.info.ECode() }
func (e *sErrNode) Slices() []ErrorInfo { return []ErrorInfo{e.info} }
func (e *sErrNode) Error() string       { return PrintDetails(e, FullPrint) }

type sErrChain struct {
	sErrNode
	next any
}

func (e *sErrChain) Error() string { return PrintDetails(e, FullPrint) }

func (e *sErrChain) Slices() []ErrorInfo {
	var c any = e
	var b []ErrorInfo
	for {
		switch n := c.(type) {
		case *sErrChain:
			b = append(b, n.info)
			c = n.next
		case *sErrNode:
			b = append(b, n.info)
			return b
		}
	}
}

// ErrNestByInfo
// Return a calm.Error which contains the given nested calm.Error, and with its top level slice set to `info`.
// If the top-level slice is the exact same as `info` and no stack trace is attached, `nested` is returned.
// If `info` is returned by calm.InfoCode and the its TCode() and ECode() match the top slice, `nested` is returned.
// Otherwise, the corresponding OnErrNest of the registered calm.IErrType will be called.
// Stack of the caller will be captured it `nested` has stack captured or OnErrNest reports true.
// This function should never fail or panic.
func ErrNestByInfo(nested Error, info ErrorInfo) Error {
	errInfo, hasTrace := _ErrExtractNestPair(nested)
	if (errInfo == info) && (!hasTrace) {
		return nested
	}
	if iCode, ok := info.(*sInfoCode); ok {
		if (iCode.TCode() == errInfo.TCode()) && (iCode.ECode() == errInfo.ECode()) {
			return nested
		}
	}
	err := &sErrChain{sErrNode: sErrNode{info: info}, next: nested}
	report := pErrOnNestSafe(info.TCode(), err)
	if hasTrace || report {
		err.trace = pWithTrace()
	}
	return err
}

func _ErrExtractNestPair(nested Error) (ErrorInfo, bool) {
	switch t := nested.(type) {
	case *sErrNode:
		return t.info, t.trace != nil
	case *sErrChain:
		return t.info, t.trace != nil
	}
	return nil, false
}

// ErrByInfo
// Create a new calm.Error with its root slice reference set to info.
// The corresponding OnErrRoot of the registered calm.IErrType will be called.
// Stack of the caller will be captured if OnErrRoot reports true.
// This function should never fail or panic.
func ErrByInfo(info ErrorInfo) Error {
	err := &sErrNode{info: info}
	if pErrOnRootSafe(err.TCode(), err) {
		err.trace = pWithTrace()
	}
	return err
}

func pWithTrace() (result []uintptr) {
	result = make([]uintptr, 2048)
	n := runtime.Callers(0, result)
	result = append([]uintptr(nil), result[:n]...)
	return
}

// Unreachable
// Mark the current return path as unreachable.
// If an Unreachable is evaluated, an EInternal will be raised
func Unreachable[T any]() T { panic(_ErrPath()) }

// Unreachable2
// Mark the current return path as unreachable.
// If an Unreachable2 is evaluated, an EInternal will be raised
func Unreachable2[T any, U any]() (T, U) { panic(_ErrPath()) }

// Unreachable3
// Mark the current return path as unreachable.
// If an Unreachable3 is evaluated, an EInternal will be raised
func Unreachable3[T any, U any, V any]() (T, U, V) { panic(_ErrPath()) }

func _ErrPath() Error { return ErrClean(EInternal, "unreachable reached") }

// Throw Raise an error by calling panic
func Throw(err error) { panic(err) }

type sInfoCode struct {
	fCode uint64
}

func (e *sInfoCode) TCode() uint32  { return uint32(e.fCode >> 32) }
func (e *sInfoCode) ECode() uint32  { return uint32(e.fCode) }
func (e *sInfoCode) Meta() any      { return nil }
func (e *sInfoCode) Detail() string { return e.Clean() }
func (e *sInfoCode) Clean() string  { return "" }

// InfoCode construct a ErrorInfo that represents an error code
func InfoCode(err uint64) ErrorInfo { return &sInfoCode{fCode: err} }

// ErrCodeN Equivalent of ErrNestByInfo(nested, InfoCode(err))
func ErrCodeN(nested Error, err uint64) Error { return ErrNestByInfo(nested, InfoCode(err)) }

// ErrCode Equivalent of ErrByInfo(InfoCode(err))
func ErrCode(err uint64) Error { return ErrByInfo(InfoCode(err)) }

// ThrowCodeN Equivalent of Throw(ErrCodeN(nested, err))
func ThrowCodeN(nested Error, err uint64) { Throw(ErrCodeN(nested, err)) }

// ThrowCode Equivalent of Throw(ErrCode(err))
func ThrowCode(err uint64) { Throw(ErrCode(err)) }

type sInfoClean struct {
	sInfoCode
	msg string
}

func (e *sInfoClean) Detail() string { return e.msg }
func (e *sInfoClean) Clean() string  { return e.msg }

// InfoClean construct a ErrorInfo that represents an error code with a sanitized message
func InfoClean(err uint64, msg string) ErrorInfo {
	return &sInfoClean{sInfoCode: sInfoCode{fCode: err}, msg: msg}
}

// ErrCleanN Equivalent of ErrNestByInfo(nested, InfoClean(err, msg))
func ErrCleanN(nested Error, err uint64, msg string) Error {
	return ErrNestByInfo(nested, InfoClean(err, msg))
}

// ErrClean Equivalent of ErrByInfo(InfoClean(err, msg))
func ErrClean(err uint64, msg string) Error { return ErrByInfo(InfoClean(err, msg)) }

// ThrowCleanN Equivalent of Throw(ErrCleanN(nested, err, msg))
func ThrowCleanN(nested Error, err uint64, msg string) { Throw(ErrCleanN(nested, err, msg)) }

// ThrowClean Equivalent of Throw(ErrClean(err, msg))
func ThrowClean(err uint64, msg string) { Throw(ErrClean(err, msg)) }

type sInfoErr struct {
	sInfoCode
	err error
}

func (e *sInfoErr) Detail() string { return e.err.Error() }
func (e *sInfoErr) Clean() string  { return e.err.Error() }
func (e *sInfoErr) Meta() any      { return e.err }

// InfoErr construct a ErrorInfo that represents an error code with a nested error
func InfoErr(err uint64, sys error) ErrorInfo {
	return &sInfoErr{sInfoCode: sInfoCode{fCode: err}, err: sys}
}

// ErrErrN Equivalent of ErrNestByInfo(nested, InfoErr(err, sys))
func ErrErrN(nested Error, err uint64, sys error) Error {
	return ErrNestByInfo(nested, InfoErr(err, sys))
}

// ErrErr construct an Error that nests the given error
func ErrErr(err uint64, nested error) Error {
	if typed, ok := nested.(Error); ok {
		return ErrNestByInfo(typed, InfoCode(err))
	}
	return ErrByInfo(InfoErr(err, nested))
}

// ThrowErrN Equivalent of Throw(ErrCleanN(nested, err, sys))
func ThrowErrN(nested Error, err uint64, sys error) { Throw(ErrErrN(nested, err, sys)) }

// ThrowErr Equivalent of Throw(ErrClean(err, sys))
func ThrowErr(err uint64, sys error) { Throw(ErrErr(err, sys)) }

type sInfoDetail struct {
	sInfoCode
	clean, detail string
}

func (e *sInfoDetail) Clean() string { return e.clean }
func (e *sInfoDetail) Meta() any     { return nil }

func (e *sInfoDetail) Detail() string {
	if e.detail != "" {
		return e.detail
	}
	return e.clean
}

// InfoDetail construct a ErrorInfo that represents an error code with a sanitized message and a full message
func InfoDetail(err uint64, clean, detail string) ErrorInfo {
	return &sInfoDetail{sInfoCode: sInfoCode{fCode: err}, clean: clean, detail: detail}
}

// ErrDetailN Equivalent of ErrNestByInfo(nested, InfoInfoDetail(err, clean, detail))
func ErrDetailN(nested Error, err uint64, clean, detail string) Error {
	return ErrNestByInfo(nested, InfoDetail(err, clean, detail))
}

// ErrDetail Equivalent of ErrByInfo(InfoDetail(err,  clean, detail))
func ErrDetail(err uint64, clean, detail string) Error {
	return ErrByInfo(InfoDetail(err, clean, detail))
}

// ThrowDetailN Equivalent of Throw(ErrDetailN(nested, err,  clean, detail))
func ThrowDetailN(nested Error, err uint64, clean, detail string) {
	Throw(ErrDetailN(nested, err, clean, detail))
}

// ThrowDetail Equivalent of Throw(ErrDetail(err,  clean, detail))
func ThrowDetail(err uint64, clean, detail string) { Throw(ErrDetail(err, clean, detail)) }

type sInfoStringer struct {
	sInfoCode
	msg  string
	meta fmt.Stringer
}

func (e *sInfoStringer) Clean() string { return e.msg }
func (e *sInfoStringer) Meta() any     { return e.meta }

func (e *sInfoStringer) Detail() (res string) {
	defer func() { _ = recover() }()
	return e.meta.String()
}

// InfoStringer construct a ErrorInfo that represents an error code with a sanitized message and a stringer object
func InfoStringer(err uint64, msg string, meta fmt.Stringer) ErrorInfo {
	return &sInfoStringer{sInfoCode: sInfoCode{fCode: err}, msg: msg, meta: meta}
}

// ErrStringerN Equivalent of ErrNestByInfo(nested, InfoStringer(err, msg, meta))
func ErrStringerN(nested Error, err uint64, msg string, meta fmt.Stringer) Error {
	return ErrNestByInfo(nested, InfoStringer(err, msg, meta))
}

// ErrStringer Equivalent of ErrByInfo(InfoDetail(err, msg, meta))
func ErrStringer(err uint64, msg string, meta fmt.Stringer) Error {
	return ErrByInfo(InfoStringer(err, msg, meta))
}

// ThrowStringerN Equivalent of Throw(ErrDetailN(nested, err, msg, meta))
func ThrowStringerN(nested Error, err uint64, msg string, meta fmt.Stringer) {
	Throw(ErrStringerN(nested, err, msg, meta))
}

// ThrowStringer Equivalent of Throw(ErrDetail(err, msg, meta))
func ThrowStringer(err uint64, msg string, meta fmt.Stringer) { Throw(ErrStringer(err, msg, meta)) }
