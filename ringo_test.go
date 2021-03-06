package ringo

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CreationTestSuite struct {
	suite.Suite
}

type WriteTestSuite struct {
	suite.Suite
	rbuf *RingBuffer
	size int
}

type ReadTestSuite struct {
	suite.Suite
	rbuf *RingBuffer
	size int
}

type SafetyTestSuite struct {
	suite.Suite
	rbuf *RingBuffer
	size int
}

type ConcurrencyTestSuite struct {
	suite.Suite
	rbuf *RingBuffer
	size int
}

func (this *CreationTestSuite) TestRingBufferCreation() {
	assert := assert.New(this.Suite.T())

	bufSize := 1500
	ringBuf := NewBuffer(bufSize)

	assert.Equal(ringBuf.Buffer.N, bufSize)
	assert.Equal(ringBuf.WritableBytes(), bufSize)
	assert.Equal(ringBuf.UnreadBytes(), 0)
}

// func TestRingBufferCreation(t *testing.T) {
// 	suite.Run(t, new(CreationTestSuite))
// }

func (this *WriteTestSuite) SetupTest() {
	this.size = 1500
	this.rbuf = NewBuffer(this.size)
}

func (this *WriteTestSuite) TestWriteWritesAppropriateLength() {
	assert := assert.New(this.Suite.T())
	bufSize := this.rbuf.Buffer.N
	stringToWrite := `This is a Test String`
	numBytesToWrite := len([]byte(stringToWrite))

	numBytesWritten, err := this.rbuf.Write([]byte(stringToWrite))
	if assert.NoError(err) {
		assert.Equal(numBytesWritten, numBytesToWrite, "Should write the correct amount to the ring buffer")
		assert.Equal(this.rbuf.WritableBytes(), bufSize-numBytesWritten, "The writable space should be less than max size by the amount written")
	}
}

func (this *WriteTestSuite) TestWriteErrorsIfWriteIsTooBig() {
	t := this.Suite.T()
	stringSize := this.size + 100
	stringToWrite := make([]byte, stringSize)
	n, err := this.rbuf.Write(stringToWrite)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Size to write exceeds the buffer size")
		assert.Equal(t, n, -1, "Write should return -1 in event of size error")
	}
}

func (this *WriteTestSuite) TestWriteMaxSizeWorks() {
	t := this.Suite.T()

	stringToWrite := makeString(this.size)
	n, err := this.rbuf.Write(stringToWrite)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, n, this.size, "Should fill the buffer")
}

func (this *WriteTestSuite) TestDoneBlocksWrite() {
	t := this.Suite.T()

	stringToWrite := makeString(150)
	n, err := this.rbuf.Write(stringToWrite)
	assert.Nil(t, err, "Write error should be nil")
	assert.Equal(t, 150, n, "Should return the appropriate number of bytes written.")

	this.rbuf.WriteIsComplete()

	n, err = this.rbuf.Write(stringToWrite)
	assert.NotNil(t, err, "Should error that the buffer has been marked 'done'")
	assert.Equal(t, "This buffer has been marked as done.  Use Reset() to reopen the buffer for writing", err.Error())
	assert.Equal(t, 0, n, "Should denote that 0 bytes were written")
}

func TestRingBufferWrite(t *testing.T) {
	suite.Run(t, new(WriteTestSuite))
}

func (this *ReadTestSuite) SetupTest() {
	this.size = 150
	this.rbuf = NewBuffer(this.size)

	this.rbuf.Write(makeString(this.size))
}

func (this *ReadTestSuite) TestReadReadsAppropriateLength() {
	t := this.Suite.T()
	readBytes := make([]byte, 25)

	n, err := this.rbuf.Read(readBytes)
	if assert.Nil(t, err) {
		assert.Equal(t, n, len(readBytes))
		assert.Equal(t, this.rbuf.UnreadBytes(), this.size-len(readBytes))
	}
}

func TestRingBufferRead(t *testing.T) {
	suite.Run(t, new(ReadTestSuite))
}

func (this *SafetyTestSuite) SetupTest() {
	this.size = 150
	this.rbuf = NewBuffer(this.size)

	this.rbuf.Write(makeString(this.size / 2))
}

func (this *SafetyTestSuite) TestIsSafeToWrite() {
	t := this.Suite.T()
	isSafe := this.rbuf.isSafeToWrite(this.size / 4)
	assert.Equal(t, isSafe, true)

	isUnsafe := this.rbuf.isSafeToWrite(this.size)
	assert.Equal(t, isUnsafe, false)
}

func (this *SafetyTestSuite) TestIsSafeToRead() {
	t := this.Suite.T()
	isSafe := this.rbuf.isSafeToRead(this.size / 4)
	assert.Equal(t, isSafe, true)

	isUnsafe := this.rbuf.isSafeToRead(this.size)
	assert.Equal(t, isUnsafe, false)

	this.rbuf.WriteIsComplete()
	unsafeIsSafeAfterDone := this.rbuf.isSafeToRead(this.size)
	assert.Equal(t, unsafeIsSafeAfterDone, true)
}

// func TestRingBufferSafety(t *testing.T) {
// 	suite.Run(t, new(SafetyTestSuite))
// }

func (this *ConcurrencyTestSuite) SetupTest() {
	this.size = 150
	this.rbuf = NewBuffer(this.size)

	this.rbuf.Write(makeString(this.size / 2))
}

func (this *ConcurrencyTestSuite) TestConcurrencySafety() {
	t := this.Suite.T()
	wg := &sync.WaitGroup{}
	done := make(chan interface{})
	timeout := make(chan interface{})

	wg.Add(1)
	go func() {
		n, err := this.rbuf.Write(makeString(this.size))
		if assert.Nil(t, err, "Write should not cause an error") {
			assert.Equal(t, n, this.size, "Should have written an amount of bytes equal to the size of the entire buffer")
		}
		wg.Done()
	}()

	go func() {
		time.Sleep(2 * time.Second)
		close(timeout)
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Error("ringBuffer wrote an amount it should not have been able to write")
	case <-timeout:
	}
}

func TestConcurrencySafety(t *testing.T) {
	suite.Run(t, new(ConcurrencyTestSuite))
}

func TestReset(t *testing.T) {
	bufSize := 150
	ringBuf := NewBuffer(bufSize)

	stringToWrite := makeString(75)

	n, err := ringBuf.Write(stringToWrite)
	assert.Nil(t, err, "No write error should have been thrown")

	n = ringBuf.WritableBytes()
	assert.Equal(t, len(stringToWrite), n)

	ringBuf.Reset()

	n = ringBuf.WritableBytes()
	assert.Equal(t, bufSize, n, "Writable bytes should be the full buffer size")
}

func TestDoneReset(t *testing.T) {
	bufSize := 150
	ringBuf := NewBuffer(bufSize)

	stringToWrite := makeString(75)

	n, err := ringBuf.Write(stringToWrite)
	assert.Nil(t, err, "No write error should have been thrown")

	ringBuf.WriteIsComplete()
	n, err = ringBuf.Write(stringToWrite)
	assert.NotNil(t, err, "Buffer should be closed for writing")
	assert.Equal(t, 0, n, "No bytes should have been written to buffer")

	ringBuf.Reset()
	n = ringBuf.WritableBytes()
	assert.Equal(t, bufSize, n, "Buffer should have maximum bytes available for writing")

	n = ringBuf.UnreadBytes()
	assert.Equal(t, 0, n, "There should be zero unread bytes after a Reset()")
}

func TestDoneCalledWithPendingWrite(t *testing.T) {
	bufSize := 150
	ringBuf := NewBuffer(bufSize)

	stringToWrite := makeString(75)

	n, err := ringBuf.Write(stringToWrite)
	assert.Nil(t, err, "No write error should have been thrown")
	assert.Equal(t, 75, n, "Should have written 75 bytes to buffer")

	go func() {
		pendingWriteString := makeString(100)
		n, err := ringBuf.Write(pendingWriteString)
		assert.Nil(t, err, "A pending write when WriteIsComplete is called should not cause an error")
		assert.Equal(t, 100, n, "Should still write if a write is pending when WriteIsComplete is called")
	}()

	goFuncStartTimer := time.NewTimer(50 * time.Millisecond)
	<-goFuncStartTimer.C
	ringBuf.WriteIsComplete()
	readSlice := make([]byte, 25)
	readBytes, err := ringBuf.Read(readSlice)
	assert.Nil(t, err, "Shouldn't cause an error to read from a Done buffer")
	assert.Equal(t, 25, readBytes, "Reading from a Done buffer should not read from the pending write")
	//Sometimes I wonder: "If I have to use a timer wait in a test, is it really concurrency safe?"
	goFuncEndTimer := time.NewTimer(50 * time.Millisecond)
	<-goFuncEndTimer.C

	unreadBytes := ringBuf.UnreadBytes()
	assert.Equal(t, 150, unreadBytes, "Should have loaded the pending write into the buffer")
}

func makeString(size int) []byte {
	iter := 0
	stringToWrite := make([]byte, size)
	for i := 0; i < size; i++ {
		iter += copy(stringToWrite[iter:], "x")
	}

	return stringToWrite
}
