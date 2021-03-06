package flyline

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// Note: The array capacity must be a power of two, e.g. 2, 4, 8, 16, 32, 64, etc.
func NewArrayBuffer(capacity int64) Buffer {
	b := &arrayBuffer{
		capacity: capacity,
		buffer:   newArray(capacity),
		wdSeq:    newCursor(),
		wpSeq:    newCursor(),
		rdSeq:    newCursor(),
		rpSeq:    newCursor(),
		sts:      &status{},
		mutex:    &sync.Mutex{},
	}
	b.sts.setRunning()
	return b
}

type arrayBuffer struct {
	capacity int64
	buffer   *array
	wpSeq    *cursor
	wdSeq    *cursor
	rpSeq    *cursor
	rdSeq    *cursor
	sts      *status
	mutex    *sync.Mutex
}

func (b *arrayBuffer) Send(i interface{}) (err error) {
	if b.sts.isClosed() {
		err = ErrBufSendClosed
		return
	}
	next := b.wpSeq.next()
	times := 10
	for {
		times--
		if next-b.capacity-b.rdSeq.get() <= 0 && next-(b.wdSeq.get()+1) == 0 {
			b.buffer.set(next, i)
			b.wdSeq.next()
			break
		}
		time.Sleep(ns100)
		if times <= 0 {
			runtime.Gosched()
			times = 10
		}
	}
	return
}

func (b *arrayBuffer) Recv() (value interface{}, active bool) {
	active = true
	if b.sts.isClosed() && b.Len() == int64(0) {
		active = false
		return
	}
	times := 10
	next := b.rpSeq.next()
	for {
		if next-b.wdSeq.get() <= 0 && next-(b.rdSeq.get()+1) == 0 {
			value = b.buffer.get(next)
			b.rdSeq.next()
			break
		}
		time.Sleep(ns100)
		if times <= 0 {
			runtime.Gosched()
			times = 10
		}
	}
	return
}

func (b *arrayBuffer) Len() (length int64) {
	length = b.wpSeq.get() - b.rdSeq.get()
	return
}

func (b *arrayBuffer) Close() (err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.sts.isClosed() {
		err = ErrBufCloseClosed
		return
	}
	b.sts.setClosed()
	return
}

func (b *arrayBuffer) Sync(ctx context.Context) (err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.sts.isRunning() {
		err = ErrBufSyncUnclosed
		return
	}
	for {
		ok := false
		select {
		case <-ctx.Done():
			ok = true
			break
		default:
			if b.Len() == int64(0) {
				ok = true
				break
			}
			time.Sleep(ms500)
		}
		if ok {
			break
		}
	}
	return
}
