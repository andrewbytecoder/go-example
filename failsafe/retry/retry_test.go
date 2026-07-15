package retry

import (
	"testing"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
)

type Connection struct {
}

type ErrConnecting struct {
	Err string
	error
}

func (e ErrConnecting) Error() string {
	return e.Err
}

func TestRetryBuild(t *testing.T) {
	// Retry on ErrConnecting up to 3 times with a 1 second delay between attempts
	retryPolicy := retrypolicy.NewBuilder[Connection]().
		HandleErrors(ErrConnecting).
		WithDelay(time.Second).
		WithMaxRetries(3).
		Build()

	failsafe.With(retrypolicy).Get("")
}
