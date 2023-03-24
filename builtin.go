package calm

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
