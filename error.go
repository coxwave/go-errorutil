// Package errorutil provides simple error wrapper for some features.
// Inspired by https://github.com/pkg/errors
//
// Currently, errorutil provides error chaining mechanism with hierachy, and auto stacktrace binding.
package errorutil

import (
	"errors"
	"fmt"
	"io"
)

var _ error = (*wrapped)(nil)
var _ fmt.Formatter = (*wrapped)(nil)

// wrapped is wrapped error with extended features
type wrapped struct {
	error
	*causer
	*traceable
}

// Is implements error interface. This makes error compare works properly.
func (w *wrapped) Is(err error) bool {
	return errors.Is(w.error, err)
}

func (w *wrapped) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			_, _ = fmt.Fprintf(f, "%s (caused by: %v)", w.Error(), w.Cause())
			return
		}
		fallthrough
	case 's', 'q':
		_, _ = io.WriteString(f, w.Error())
	}
}

// Wrap wraps the error with provided options.
func Wrap(err error, opts ...WrapOpt) error {
	if err == nil {
		return nil
	}
	var we *wrapped
	if errors.As(err, &we) && len(opts) == 0 {
		// If error is already wrapped, and no additional options provided, just return it
		return we
	}

	w := &wrapped{error: err}
	for _, opt := range opts {
		opt(w)
	}
	if w.traceable == nil {
		// Auto bind stack trace if not already set
		AutoStackTrace()(w)
	}

	return w
}

type WrapOpt func(w *wrapped)

// AutoStackTrace is the option which automatically bind caller's stacktrace to error. This makes some error-capturing module (like https://github.com/getsentry/sentry-go) can extract proper stacktrace of your error.
// For convenience, this option is enabled by default even if you don't include it.
func AutoStackTrace() WrapOpt {
	return func(w *wrapped) {
		w.traceable = traceableFromCallers(4)
	}
}

// FromCause is the option which wraps the error with provided cause. If you Unwrap this error, provided cause will be extracted.
func FromCause(cause error) WrapOpt {
	return func(w *wrapped) {
		w.causer = &causer{cause: cause}
	}
}

// OriginalErr returns the original error without any wrapping
func OriginalErr(err error) error {
	if err == nil {
		return nil
	}

	var we *wrapped
	if !errors.As(err, &we) {
		return err
	}

	return we.error
}
