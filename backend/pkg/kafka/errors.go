package kafka

import "errors"

// permanentError marks a handler failure that cannot succeed on retry, such as a
// malformed payload or a rejected domain state. The consumer sends these straight
// to the dead-letter topic instead of burning the retry budget.
type permanentError struct {
	err error
}

func (e permanentError) Error() string { return e.err.Error() }

func (e permanentError) Unwrap() error { return e.err }

// Permanent wraps err so the consumer treats it as non-retryable. It is used by
// message adapters to classify domain failures that will never become valid.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

// IsPermanent reports whether err (or anything it wraps) was marked Permanent.
func IsPermanent(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}

// PermanentIf wraps err as permanent when it matches any of the supplied sentinel
// errors. It keeps handler classification readable: a transient failure that
// happens to be one of the listed sentinels is treated as non-retryable.
func PermanentIf(err error, sentinels ...error) error {
	if err == nil {
		return nil
	}
	for _, s := range sentinels {
		if errors.Is(err, s) {
			return Permanent(err)
		}
	}
	return err
}
