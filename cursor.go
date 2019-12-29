package flyline

import (
	"runtime"
	"sync/atomic"
	"time"
)

func newCursor() (c *cursor) {
	c = &cursor{no: int64(-1)}
	return
}

type cursor struct {
	no int64
	rhs     [padding7]int64
}

func (c *cursor) get() (no int64) {
	no = atomic.LoadInt64(&c.no)
	return
}

func (c *cursor) next() (no int64) {
	times := retry10Times
	for {
		times--
		next := c.get() + 1
		ok := atomic.CompareAndSwapInt64(&c.no, c.no, next)
		if ok {
			no = next
			break
		}
		time.Sleep(ns100)
		if times <= 0 {
			times = retry10Times
			runtime.Gosched()
		}
	}
	return
}
