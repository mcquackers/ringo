package ringo

import (
	"errors"

	"github.com/glycerine/rbuf"
	"github.com/golang/glog"
)

type RingBuffer struct {
	Buffer  *rbuf.FixedSizeRingBuf
	diff    int
	maxSize int
}

func NewBuffer(size int) *RingBuffer {
	return &RingBuffer{
		Buffer:  rbuf.NewFixedSizeRingBuf(size),
		diff:    0,
		maxSize: size,
	}
}

func (this *RingBuffer) Write(p []byte) (int, error) {
	if this.isNotSafeToWrite(len(p)) {
		return 0, errors.New("Will cause overwrite")
	}
	n, err := this.Buffer.Write(p)
	if err != nil {
		return n, err
	}

	this.diff = this.diff + n

	return n, err
}

func (this *RingBuffer) Read(p []byte) (int, error) {
	glog.Info("Attempting to read")

	n, err := this.Buffer.Read(p)
	if err != nil {
		return n, err
	}

	this.diff = this.diff - n

	return n, err
}

func (this *RingBuffer) UnreadBytes() int {
	return this.diff
}

func (this *RingBuffer) WritableBytes() int {
	return this.maxSize - this.diff
}

func (this *RingBuffer) isNotSafeToWrite(len int) bool {
	return !(this.diff+len < this.maxSize)
}
