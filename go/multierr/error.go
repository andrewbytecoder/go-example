package multierr

import "go.uber.org/multierr"

// wrapper for multi error

func Combine(errs ...error) error {
	return multierr.Combine(errs...)
}

func Append(left error, right error) error {
	return multierr.Append(left, right)
}
