package approvalsock

import (
	"sync/atomic"
)

const reportQueueSize = 32

type reporter struct {
	queue   chan error
	dropped atomic.Uint64
}

func newReporter() *reporter {
	return &reporter{queue: make(chan error, reportQueueSize)}
}

func (r *reporter) send(err error) {
	select {
	case r.queue <- err:
	default:
		r.dropped.Add(1)
	}
}

func (r *reporter) deliver(report func(error)) {
	for {
		if count := r.dropped.Swap(0); count > 0 {
			report(&ReportsDroppedError{Count: count})
		}
		err, open := <-r.queue
		if !open {
			return
		}
		report(err)
	}
}
