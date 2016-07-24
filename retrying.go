package retrying

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"time"

	"github.com/hashicorp/go-multierror"
)

// errors
var (
	ErrTimeout             = fmt.Errorf("timeout error")
	ErrNoFunctionSpecified = fmt.Errorf("no function is specified")
)

const (
	defaultStackSize       = 4096
	defaultMaxAttemptTimes = 1
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

// can be mocked out for test
var sleep = time.Sleep

// Retryable model consisting of retry options
type Retryable struct {
	stackSize     int
	allGoroutines bool

	maxAttemptTimes int
	maxDelay        time.Duration

	waitFixed                    time.Duration
	waitRandomMin, waitRandomMax time.Duration

	f func() error

	errors []error
}

// New create new retry
func New() *Retryable {
	return &Retryable{
		stackSize:       defaultStackSize,
		maxAttemptTimes: defaultMaxAttemptTimes,
		f:               func() error { return ErrNoFunctionSpecified },
	}
}

// Stack set stack parameters used in runtime.Stack
func (r *Retryable) Stack(n int, all bool) *Retryable {
	if n <= 0 {
		r.errors = append(r.errors, fmt.Errorf("stack size must be positive integer"))
	}
	r.stackSize = n
	r.allGoroutines = all
	return r
}

// MaxAttemptTimes set max attempt times
func (r *Retryable) MaxAttemptTimes(n int) *Retryable {
	if n <= 0 {
		r.errors = append(r.errors, fmt.Errorf("max attempt times must be positive integer"))
	}
	r.maxAttemptTimes = n
	return r
}

// MaxDelay set max delay duration
func (r *Retryable) MaxDelay(d time.Duration) *Retryable {
	if d <= 0 {
		r.errors = append(r.errors, fmt.Errorf("max delay must be positive duration"))
	}
	r.maxDelay = d
	return r
}

// WaitFixed set fixed wait duration
func (r *Retryable) WaitFixed(d time.Duration) *Retryable {
	if d <= 0 {
		r.errors = append(r.errors, fmt.Errorf("wait fixed must be positive duration"))
	}
	r.waitFixed = d
	return r
}

// WaitRandom set min/max random
func (r *Retryable) WaitRandom(min, max time.Duration) *Retryable {
	if min < 0 || max < 0 {
		r.errors = append(r.errors, fmt.Errorf("wait random min/max must be positive duration"))
	}
	if min >= max {
		r.errors = append(r.errors, fmt.Errorf("wait random min must be smaller than max"))
	}
	r.waitRandomMin, r.waitRandomMax = min, max
	return r
}

// Function set function
// i should be a function with no output or last output should be an error
func (r *Retryable) Function(i interface{}) *Retryable {
	typ := reflect.TypeOf(i)
	if kind := typ.Kind(); kind != reflect.Func {
		r.errors = append(r.errors, fmt.Errorf("expected type %v but get %v", reflect.Func, kind))
		return r
	}
	if n := typ.NumIn(); n != 0 {
		r.errors = append(r.errors, fmt.Errorf("expected 0 inputs but get %v", n))
	}
	if n := typ.NumOut(); n > 0 && !typ.Out(n-1).Implements(errorInterface) {
		r.errors = append(r.errors, fmt.Errorf("expected 0 output or last output implements error interface"))
	}

	val := reflect.ValueOf(i)
	switch typ.NumOut() {
	case 0:
		r.f = r.wrapRecoverFunc(func() error {
			val.Call(nil)
			return nil
		})
	default:
		r.f = r.wrapRecoverFunc(func() error {
			outputs := val.Call(nil)
			lastOutput := outputs[len(outputs)-1]
			if lastOutput.IsNil() {
				return nil
			}
			return lastOutput.Interface().(error)
		})
	}

	return r
}

// Try call the wrap function with retry options
func (r *Retryable) Try() error {
	errors := multierror.Append(nil, r.errors...)

	// stop if errors occur in initialization
	if err := errors.ErrorOrNil(); err != nil {
		return err
	}

	// try with or without timeout
	if r.maxDelay > 0 {
		return r.tryWithTimeout()
	}
	return r.tryWithoutTimeout()
}

// helpers
//
func (r *Retryable) wrapRecoverFunc(f func() error) func() error {
	return func() (err error) {
		defer func() {
			if e := recover(); e != nil {
				buf := make([]byte, r.stackSize)
				runtime.Stack(buf, r.allGoroutines)
				err = fmt.Errorf("%v\n%s\n", e, buf)
			}
		}()

		return f()
	}
}

func (r *Retryable) wait() {
	duration := r.waitFixed
	if duration <= 0 && r.waitRandomMax > r.waitRandomMin {
		duration = r.waitRandomMin + time.Duration(rand.Int63n(int64(r.waitRandomMax-r.waitRandomMin)))
	}
	sleep(duration)
}

func (r *Retryable) tryWithTimeout() error {
	errors := &multierror.Error{}
	errChan := make(chan error)
	timer := time.NewTimer(r.maxDelay)
	count := r.maxAttemptTimes

	go func() {
		for ; count > 0; count-- {
			errChan <- r.f()
			r.wait()
		}
	}()

	for {
		select {
		case err := <-errChan:
			errors = multierror.Append(errors, err)

			if err == nil {
				return nil
			}

			if count <= 0 {
				return errors.ErrorOrNil()
			}
		case <-timer.C:
			return ErrTimeout
		}
	}
}

func (r *Retryable) tryWithoutTimeout() error {
	errors := &multierror.Error{}

	for count := r.maxAttemptTimes; count > 0; count-- {
		err := r.f()
		errors = multierror.Append(errors, err)

		if err == nil {
			return nil
		}

		r.wait()
	}

	return errors.ErrorOrNil()
}
