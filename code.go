package calm

import (
	"sync"
	"sync/atomic"
)

const (
	// EConfig A configuration error is detected
	EConfig = 1
	// ERequest Request is not able to be served due to procedure prerequisites not satisfied
	ERequest = 2
	// EDenied The requesting party is not granted the access to this resource, or the grant has expired
	EDenied = 3
	// EInternal The request cannot be completed due to an unspecified internal error
	EInternal = 4

	// EStgNone Unable to find given storage record
	EStgNone = 64
	// EStgFull Unable to complete a storage write request due to short of storage quota
	EStgFull = 65
	// EStgQueue Storage data backlog too long
	EStgQueue = 66
	// EStgLost Storage handle lost due to max timeout exceeded during operation
	EStgLost = 67
	// EStgFail Generic storage failure
	EStgFail = 68

	// ENetNone Unable to find give data interface/connection
	ENetNone = 96
	// ENetEarly Data interface/connection not ready
	ENetEarly = 97
	// ENetDown Data interface/connection is down
	ENetDown = 98
	// ENetQueue Data interface/connection backlog too long
	ENetQueue = 99
	// ENetRetry Data request is unable to be completed at the time, retry by user is suggested
	ENetRetry = 100
	// ENetMaxRetry Data request is unable to be completed after max configured automatic retries
	ENetMaxRetry = 101
	// ENetLost Data interface/connection max timeout exceeded
	ENetLost = 102
	// ENetFail Generic data interface/connection failure
	ENetFail = 103

	// EResNone Requested resource not found
	EResNone = 128
	// EResAuth Resource requested authorization not satisfied
	EResAuth = 129
	// EResRetry Resource not available at this time. User retry is suggested
	EResRetry = 130
	// EResGone Resource permanently removed
	EResGone = 131
	// EResFail Resource generic failure
	EResFail = 132

	// ETimeout Action timeout exceeded
	ETimeout = 512
	// ECancel Action cancelled
	ECancel = 513
	// EBacklog Task backlog limit exceeded
	EBacklog = 514
)

func AsSys(code uint32) func(Error) { return func(err Error) { ThrowErr(uint64(code), err) } }

var (
	AsEConfig   = AsSys(EConfig)
	AsERequest  = AsSys(ERequest)
	AsEDenied   = AsSys(EDenied)
	AsEInternal = AsSys(EInternal)

	AsEStgNone  = AsSys(EStgNone)
	AsEStgFull  = AsSys(EStgFull)
	AsEStgQueue = AsSys(EStgQueue)
	AsEStgLost  = AsSys(EStgLost)
	AsEStgFail  = AsSys(EStgFail)

	AsENetNone     = AsSys(ENetNone)
	AsENetEarly    = AsSys(ENetEarly)
	AsENetDown     = AsSys(ENetDown)
	AsENetQueue    = AsSys(ENetQueue)
	AsENetRetry    = AsSys(ENetRetry)
	AsENetMaxRetry = AsSys(ENetMaxRetry)
	AsENetLost     = AsSys(ENetLost)
	AsENetFail     = AsSys(ENetFail)

	AsEResNone  = AsSys(EResNone)
	AsEResAuth  = AsSys(EResAuth)
	AsEResRetry = AsSys(EResRetry)
	AsEResGone  = AsSys(EResGone)
	AsEResFail  = AsSys(EResFail)

	AsETimeout = AsSys(ETimeout)
	AsECancel  = AsSys(ECancel)
	AsEBacklog = AsSys(EBacklog)
)

var _ErrSysMsg = map[uint32]string{
	EConfig:   "configuration error",
	ERequest:  "invalid request",
	EDenied:   "access denied",
	EInternal: "internal server error",

	EStgNone:  "storage: not found",
	EStgFull:  "storage: quota/space full",
	EStgQueue: "storage: sys backlog",
	EStgLost:  "storage: sys timeout",
	EStgFail:  "storage: fail",

	ENetNone:     "com: not found",
	ENetEarly:    "com: not ready",
	ENetDown:     "com: down",
	ENetQueue:    "com: backlog",
	ENetRetry:    "com: retry suggested",
	ENetMaxRetry: "com: max retry reached",
	ENetLost:     "com: dev timeout",
	ENetFail:     "com: dev fail",

	EResNone:  "resource: not found",
	EResAuth:  "resource: denied",
	EResRetry: "resource: retry suggested",
	EResGone:  "resource: removed",
	EResFail:  "resource: fail",

	ETimeout: "sys: action timeout exceeded",
	ECancel:  "sys: action cancelled",
	EBacklog: "sys: task backlog exceeded",
}

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
	// DefaultMsg get a default clean message for a specific code
	DefaultMsg(err uint32) string
}

type _ErrSys struct{}

func (e *_ErrSys) Type() uint32              { return 0 }
func (e *_ErrSys) OnErrNest(err Error) bool  { return err.ECode() == EInternal }
func (e *_ErrSys) OnErrRoot(err Error) bool  { return err.ECode() == EInternal }
func (e *_ErrSys) ErrName(err uint32) string { return _ErrSysMsg[err] }
func (e *_ErrSys) DefaultMsg(uint32) string  { return "" }

type _ErrNil struct{ uint32 }

func (e _ErrNil) Type() uint32             { return e.uint32 }
func (e _ErrNil) OnErrNest(Error) bool     { return false }
func (e _ErrNil) OnErrRoot(Error) bool     { return false }
func (e _ErrNil) ErrName(uint32) string    { return "" }
func (e _ErrNil) DefaultMsg(uint32) string { return "" }

var (
	_Reg = sync.Map{}
	_Cnt = atomic.Uint32{}
)

func AddErrType(supply func(id uint32) IErrType) uint32 {
	tCode := _Cnt.Add(1)
	_Reg.Store(tCode, supply(tCode))
	return tCode
}

func MakeErrCode(typeCode uint32, errCode uint32) uint64 {
	return (uint64(typeCode) << uint64(32)) | uint64(errCode)
}

func fGetErrType(tCode uint32) IErrType {
	if reg, ok := _Reg.Load(tCode); ok {
		return reg.(IErrType)
	}
	return _ErrNil{uint32: tCode}
}

func pErrOnNestSafe(tCode uint32, err Error) (res bool) {
	defer func() { _ = recover() }()
	res = fGetErrType(tCode).OnErrNest(err)
	return
}

func pErrOnRootSafe(tCode uint32, err Error) (res bool) {
	defer func() { _ = recover() }()
	res = fGetErrType(tCode).OnErrRoot(err)
	return
}

func pErrDefaultMsgSafe(tCode uint32, eCode uint32) (res string) {
	defer func() { _ = recover() }()
	res = fGetErrType(tCode).DefaultMsg(eCode)
	return
}

func init() {
	_Reg.Store(uint32(0), &_ErrSys{})
}
