package ui

import (
	"context"
	"sync"
	"time"
)

// A tool for displaying the status of slow operations in the UI.
// If some operation executes for more than some minimal period,
// we have to display some changes in UI to avoid window "freezeng".
// But, if we show these changes every time of usage such opreation,
// widgets change their states too often (if the operation finished momentally),
// and it causes the feeling of screen "blinking".
// To avoid it we have to implement such a logick:
//   - Until some very short period of operation running
//     we do not change UI state (for the case when operation will finish
//     in a few moments).
//   - When some timout is reached, we show in UI that our program
//     didn't get stuck, but it needs more time to respond ("loading" state).
//   - When the operation finished, we wait during some period only if we just
//     showed "loading" state (again, to avoid widgets "blinking"). For it,
//     we need a minimum time period of displaying "loading" state.
//
// By this way, we can avoid "blinking" widgets and "freezing" window.
type indicatedSlowOperation struct {
	timeoutToIndicateLoading     time.Duration
	minPeriodOfLoadingIndication time.Duration
	indicateLoading              func()
	indicateLoadingFinished      func()

	outerCtx context.Context
	outerWG  *sync.WaitGroup

	parallelCallsProtection  sync.Mutex
	stopWaitingTimeout       func()
	recursiveCallsProtection uint8

	loadingIndicationLocker               sync.Mutex
	indicatingLoading                     bool
	earliestMomentToStopLoadingIndication time.Time
}

// indicateLoading - will be called in a new goroutine.
// This goroutine uses wg to wait for its' exit
// and ctx to cancel waiting for timout of operation executing.
func newSlowOperation(
	ctx context.Context,
	wg *sync.WaitGroup,
	timeoutToIndicateLoading time.Duration,
	minPeriodOfLoadingIndication time.Duration,
	indicateLoading func(),
	indicateLoadingFinished func(),
) *indicatedSlowOperation {
	res := &indicatedSlowOperation{
		timeoutToIndicateLoading:     timeoutToIndicateLoading,
		minPeriodOfLoadingIndication: minPeriodOfLoadingIndication,
		indicateLoading:              indicateLoading,
		indicateLoadingFinished:      indicateLoadingFinished,
		outerCtx:                     ctx,
		outerWG:                      wg,
	}

	return res
}

// Gorotine-safe. Can be called several times before LeaveAndWait() call (for recursive cases).
// After timeoutToIndicateLoading period expiration since the first call of Enter() method,
// indicateLoading func will be called in a new goroutine (if LeaveAndWait() wasn't called earlier).
// It can lock current goroutine if at the moment other gouroutine is waiting for expiration of
// minPeriodOfLoadingIndication period by calling LeaveAndWait().
func (o *indicatedSlowOperation) Enter() {
	var ctx context.Context

	o.parallelCallsProtection.Lock()

	begin := o.recursiveCallsProtection == 0

	o.recursiveCallsProtection++

	if begin {
		ctx, o.stopWaitingTimeout = context.WithCancel(o.outerCtx)
	}

	o.parallelCallsProtection.Unlock()

	if !begin {
		return
	}

	o.outerWG.Add(1)

	go func() {
		defer o.outerWG.Done()

		select {
		case <-ctx.Done():
		case <-time.After(o.timeoutToIndicateLoading):
			o.loadingIndicationLocker.Lock()
			defer o.loadingIndicationLocker.Unlock()

			//ctx.Err() == nil check is important: it works in cases
			//when the context was cancelled in the same moment with timer expiration.
			//Without this check loading state can be applied like in the case when
			//LeaveAndWait() was called before Enter().
			if !o.indicatingLoading && ctx.Err() == nil {
				o.indicateLoading()

				o.earliestMomentToStopLoadingIndication = time.Now().Add(o.minPeriodOfLoadingIndication)

				o.indicatingLoading = true
			}
		}
	}()
}

// Goroutine-safe. Should be called same times as Enter() was called before to cause
// indicateLoadingFinished func call. Causes indicateLoadingFinished func call (in current
// goroutine) only if timeoutToIndicateLoading was reached (in the opposite case,
// indicateLoading func won't be called and indicateLoadingFinished too). Locks
// current goroutine until minPeriodOfLoadingIndication since the expiration of
// indicateLoading func call.
func (o *indicatedSlowOperation) LeaveAndWait() {
	o.parallelCallsProtection.Lock()

	o.recursiveCallsProtection--

	begin := o.recursiveCallsProtection == 0

	if begin {
		o.stopWaitingTimeout()
	}

	o.parallelCallsProtection.Unlock()

	if !begin {
		return
	}

	o.loadingIndicationLocker.Lock()
	defer o.loadingIndicationLocker.Unlock()

	if o.indicatingLoading {
		timeToWait := time.Until(o.earliestMomentToStopLoadingIndication)

		if timeToWait > 0 {
			select {
			case <-o.outerCtx.Done():
			case <-time.After(timeToWait):
				o.indicateLoadingFinished()

				o.indicatingLoading = false
			}
		}
	}
}
