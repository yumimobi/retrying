package retrying

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
)

func TestStack(t *testing.T) {
	r := New().Stack(-1, false)
	if len(r.errors) != 1 {
		t.Error("number of errors should be 1")
	}
}

func TestMaxAttemptTimes(t *testing.T) {
	r := New().MaxAttemptTimes(-1)
	if len(r.errors) != 1 {
		t.Error("number of errors should be 1")
	}
}

func TestMaxDelay(t *testing.T) {
	r := New().MaxDelay(time.Duration(0))
	if len(r.errors) != 1 {
		t.Error("number of errors should be 1")
	}
}

func TestWaitFixed(t *testing.T) {
	r := New().WaitFixed(time.Duration(0))
	if len(r.errors) != 1 {
		t.Error("number of errors should be 1")
	}
}

func TestWaitRandom(t *testing.T) {
	r1 := New().WaitRandom(time.Duration(-1), time.Second)
	if len(r1.errors) != 1 {
		t.Error("number of errors should be 1")
	}

	r2 := New().WaitRandom(time.Minute, time.Second)
	if len(r2.errors) != 1 {
		t.Error("number of errors should be 1")
	}

	r3 := New().WaitRandom(time.Duration(-1), time.Duration(-1))
	if len(r3.errors) != 2 {
		t.Error("number of errors should be 2")
	}
}

func TestFunction(t *testing.T) {
	r1 := New().Function(1)
	if len(r1.errors) != 1 {
		t.Error("number of errors should be 1")
	}

	r2 := New().Function(func(_ int) int { return 1 })
	if len(r2.errors) != 2 {
		t.Error("number of errors should be 2")
	}
}

func TestTry(t *testing.T) {
	// stop due to errors in initialization
	if err := New().Function(func(_ int) {}).Try(); err == nil {
		t.Error("error should not be nil")
	}

	// no function specified
	errors := multierror.Append(nil, ErrNoFunctionSpecified)
	if err := New().Try(); err.Error() != errors.Error() {
		t.Errorf("error should be no function specified but get %v", err)
	}

	// no panic
	if err := New().Function(func() {}).Try(); err != nil {
		t.Errorf("error should be nil but get %v", err)
	}

	// succeed after two panics
	c1 := 5
	if err := New().MaxAttemptTimes(5).Function(func() {
		c1--
		if c1 == 2 {
			return
		}
		panic("panic here")
	}).Try(); err != nil {
		t.Errorf("error should be nil but get %v", err)
	}

	// succeed after two errors
	c2 := 5
	if err := New().MaxAttemptTimes(5).
		Function(func() error {
			c2--
			if c2 == 2 {
				return nil
			}
			return fmt.Errorf("")
		}).
		WaitRandom(time.Second, time.Second*time.Duration(2)).
		Try(); err != nil {
		t.Errorf("error should be nil but get %v", err)
	}

	// timeout
	if err := New().MaxDelay(time.Second).
		Function(func() {
			time.Sleep(time.Minute)
		}).
		Try(); err != ErrTimeout {
		t.Errorf("error should be timeout but get %v", err)
	}

	// succeed before timeout
	if err := New().MaxDelay(time.Minute).
		Function(func() {}).
		Try(); err != nil {
		t.Errorf("error should be nil but get %v", err)
	}

	// fail before timeout
	if err := New().MaxDelay(time.Minute).
		Function(func() { panic("DLLM") }).
		Try(); err == nil {
		t.Errorf("error should not be nil")
	}
}
