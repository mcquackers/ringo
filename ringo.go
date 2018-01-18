package ringo

import (
	"errors"
	"sync"

	"github.com/glycerine/rbuf"
)

type RingBuffer struct {
	Buffer             *rbuf.FixedSizeRingBuf
	readLock           *sync.Cond
	writeLock          *sync.Cond
	bufLock            *sync.Mutex
	desiredWriteLength int
	desiredWriteLock   *sync.Mutex
	diff               int
	maxSize            int
	done               bool
}

func NewBuffer(size int) *RingBuffer {
	return &RingBuffer{
		Buffer:             rbuf.NewFixedSizeRingBuf(size),
		readLock:           &sync.Cond{L: &sync.Mutex{}},
		writeLock:          &sync.Cond{L: &sync.Mutex{}},
		bufLock:            &sync.Mutex{},
		desiredWriteLength: 0,
		desiredWriteLock:   &sync.Mutex{},
		diff:               0,
		maxSize:            size,
		done:               false,
	}
}

func (this *RingBuffer) Write(p []byte) (int, error) {
	if len(p) > this.maxSize {
		return -1, errors.New("Size to write exceeds the buffer size")
	}

	this.writeLock.L.Lock()
	this.desiredWriteLock.Lock()
	this.desiredWriteLength = len(p)
	this.desiredWriteLock.Unlock()

	for !this.isSafeToWrite(len(p)) {
		this.writeLock.Wait()
	}

	n, err := this.Buffer.Write(p)
	if err != nil {
		this.writeLock.L.Unlock()
		return n, err
	}
	this.writeLock.L.Unlock()

	this.bufLock.Lock()
	this.diff = this.diff + n
	this.bufLock.Unlock()

	this.desiredWriteLock.Lock()
	this.desiredWriteLength = 0
	this.desiredWriteLock.Unlock()
	this.readLock.Signal()

	return n, err
}

func (this *RingBuffer) Read(p []byte) (int, error) {
	this.readLock.L.Lock()
	for !this.isSafeToRead(len(p)) {
		this.readLock.Wait()
	}
	n, err := this.Buffer.Read(p)
	if err != nil {
		return n, err
	}
	this.readLock.L.Unlock()

	this.bufLock.Lock()
	this.diff = this.diff - n

	this.desiredWriteLock.Lock()
	if this.maxSize-this.diff >= this.desiredWriteLength {
		this.writeLock.Signal()
	}
	this.desiredWriteLock.Unlock()
	this.bufLock.Unlock()

	return n, err
}

func (this *RingBuffer) UnreadBytes() int {
	this.bufLock.Lock()
	defer this.bufLock.Unlock()
	return this.diff
}

func (this *RingBuffer) WritableBytes() int {
	this.bufLock.Lock()
	defer this.bufLock.Unlock()
	return this.maxSize - this.diff
}

func (this *RingBuffer) isSafeToWrite(len int) bool {
	this.bufLock.Lock()
	defer this.bufLock.Unlock()
	return this.diff+len <= this.maxSize
}

func (this *RingBuffer) isSafeToRead(len int) bool {
	this.bufLock.Lock()
	defer this.bufLock.Unlock()
	return this.done || this.diff >= len
}

func (this *RingBuffer) WriteIsComplete() {
	this.bufLock.Lock()
	defer this.bufLock.Unlock()
	this.done = true
	this.readLock.Signal()
}

func (this *RingBuffer) Reset() {
	this.bufLock.Lock()
	defer this.bufLock.Unlock()
	this.diff = 0
	this.done = false
}
