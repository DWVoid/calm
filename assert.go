package calm

import "errors"

func Assert(condition bool, message ...string) Result {
	if !condition {
		if len(message) >= 1 {
			return ErrResult(errors.New(message[0]))
		} else {
			return ErrResult(errors.New(message[0]))
		}
	}
	return ValResult()
}
