package calm

import (
	"sync"
	"sync/atomic"
)

const (
	EConfig   = 1
	ERequest  = 2
	EDenied   = 3
	EInternal = 4
)

var (
	AsEConfig   = func(err Error) { ThrowNested(EConfig, err) }
	AsERequest  = func(err Error) { ThrowNested(ERequest, err) }
	AsEDenied   = func(err Error) { ThrowNested(EDenied, err) }
	AsEInternal = func(err Error) { ThrowNested(EInternal, err) }
)

type IErrType interface {
	Type() uint32
	// OnErrNest callback on a nested error is created
	// err: the nested error
	// return: if stacktrace should be included
	OnErrNest(err Error) bool
	// OnErrRoot callback on a new error is created
	// err: the newly created error
	// return: if stacktrace should be included
	OnErrRoot(err Error) bool
	// ErrName : name tag for a specific error code
	ErrName(err uint32) string
	// DefaultCleanMsg get a default clean message for a specific code
	DefaultCleanMsg(err uint32) string
}

type _ErrSys struct {
}

func (e *_ErrSys) Type() uint32 { return 0 }

func (e *_ErrSys) OnErrNest(err Error) bool { return err.ECode() == EInternal }

func (e *_ErrSys) OnErrRoot(err Error) bool { return err.ECode() == EInternal }

func (e *_ErrSys) ErrName(err uint32) string {
	switch err {
	case EConfig:
		return "configuration error"
	case ERequest:
		return "invalid request"
	case EDenied:
		return "access denied"
	case EInternal:
		return "internal server error"
	}
	return Unreachable[string]()
}

func (e *_ErrSys) DefaultCleanMsg(uint32) string { return "" }

var (
	_Reg = sync.Map{}
	_Cnt = atomic.Uint32{}
)

func AddErrType(supply func(id uint32) IErrType) uint32 {
	id := _Cnt.Add(1)
	_Reg.Store(id, supply(id))
	return id
}

func MakeErrCode(typeCode uint32, errCode uint32) uint64 {
	return (uint64(typeCode) << uint64(32)) | uint64(errCode)
}

func init() {
	_Reg.Store(0, &_ErrSys{})
}
