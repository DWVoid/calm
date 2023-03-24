package calm

type Error interface {
	Type() int
	error
}

type errNested struct {
	errType   int
	errNested error
}

func (s errNested) Type() int {
	return s.errType
}

func (s errNested) Error() string {
	return s.errNested.Error()
}

type errMessage struct {
	errType    int
	errMessage string
}

func (s errMessage) Type() int {
	return s.errType
}

func (s errMessage) Error() string {
	return s.errMessage
}

func NestedError(err int, nested error) Error {
	switch typed := nested.(type) {
	case errNested:
		return errNested{errType: err, errNested: typed.errNested}
	default:
		return errNested{errType: err, errNested: nested}
	}
}

func TaggedError(err int, message string) Error {
	return errMessage{errType: err, errMessage: message}
}

func Unreachable[T any]() (ret T) {
	return
}

func Throw(err error) {
	panic(err)
}

func ThrowNested(err int, nested error) {
	Throw(NestedError(err, nested))
}

func ThrowTagged(err int, message string) {
	Throw(TaggedError(err, message))
}
