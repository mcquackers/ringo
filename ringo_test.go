package ringo

import (
	"sync"
	"testing"

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

// func TestRingBufferWrite(t *testing.T) {
// 	suite.Run(t, new(WriteTestSuite))
// }

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

// func TestRingBufferRead(t *testing.T) {
// 	suite.Run(t, new(ReadTestSuite))
// }

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
	wg.Add(1)
	go func() {
		go func() {
			n, err := this.rbuf.Write(makeString(this.size / 3))
			if assert.Nil(t, err, "Write should not cause an error") {
				assert.Equal(t, n, this.size/3, "Should have written an amount of bytes equal to one third the size of the entire buffer")
			}
		}()

		n, err := this.rbuf.Write(makeString(this.size))
		if assert.Nil(t, err, "Write should not cause an error") {
			assert.Equal(t, n, this.size, "Should have written an amount of bytes equal to the size of the entire buffer")
		}
		wg.Done()
	}()

	bytesToRead := make([]byte, this.rbuf.UnreadBytes())
	n, err := this.rbuf.Read(bytesToRead)
	wg.Wait()
	if assert.Nil(t, err) {
		assert.Equal(t, len(bytesToRead), n)
		assert.Equal(t, this.size/3, this.rbuf.desiredWriteLength, "Should want to write an amount of bytes equal to one third the size of the buffer")
		assert.Equal(t, this.rbuf.UnreadBytes(), this.size, "Should be a completely full buffer")
		assert.Equal(t, this.rbuf.WritableBytes(), 0, "Should have no more space to write in the buffer")
	}

}

func TestConcurrencySafety(t *testing.T) {
	suite.Run(t, new(ConcurrencyTestSuite))
}

func makeString(size int) []byte {
	iter := 0
	stringToWrite := make([]byte, size)
	for i := 0; i < size; i++ {
		iter += copy(stringToWrite[iter:], "x")
	}

	return stringToWrite
}
