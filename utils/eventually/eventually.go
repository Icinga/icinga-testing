package eventually

import (
	"github.com/stretchr/testify/require"
	"runtime"
	"sync"
	"time"
)

// Assert repeatedly calls a sub-test given by f until it succeeds. The frequency of calls is determined by interval.
// Failures in the sub-test do not let the surrounding test fail, unless the duration given by timeout passed. In this
// case, a final attempt is made that will also make the surrounding test fail so that the output of the final attempt
// is shown in the overall test output.
func Assert(t require.TestingT, f func(t require.TestingT), timeout time.Duration, interval time.Duration) bool {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ch := make(chan bool, 1)

	for tick := ticker.C; ; {
		select {
		case <-timer.C:
			// Timeout expired, final call to f that forwards errors to the real TestingT.
			fakeT := &wrapT{t: t}
			f(fakeT)
			return !fakeT.Failed()
		case <-tick:
			tick = nil
			go func() {
				fakeT := new(wrapT)
				defer func() { // use defer as f may do runtime.Goexit()
					ch <- fakeT.Failed()
				}()
				f(fakeT)
			}()
		case failed := <-ch:
			if !failed {
				return true
			}
			tick = ticker.C
		}
	}
}

// Require performs the same test as Assert but calls t.FailNow() on failure.
func Require(t require.TestingT, f func(t require.TestingT), timeout time.Duration, interval time.Duration) {
	if !Assert(t, f, timeout, interval) {
		t.FailNow()
	}
}

// wrapT is a helper type for Assert/Require that implements the require.TestingT interface and is  able to intercept
// calls that would make the test fail and record the fact that such a call happened. Optionally, another
// require.TestingT can be given, to which all calls are forwarded.
type wrapT struct {
	mutex  sync.Mutex
	failed bool
	t      require.TestingT
}

// Errorf wraps the same function from testing.T/require.TestingT.
func (f *wrapT) Errorf(format string, args ...interface{}) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.failed = true

	if f.t != nil {
		f.t.Errorf(format, args...)
	}
}

// FailNow wraps the same function from testing.T/require.TestingT.
func (f *wrapT) FailNow() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.failed = true

	if f.t != nil {
		f.t.FailNow()
	} else {
		runtime.Goexit()
	}
}

// Failed returns whether any function was called that made the test case fail.
func (f *wrapT) Failed() bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.failed
}

var _ require.TestingT = (*wrapT)(nil)
