package bytechunk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// WriteBytes writes bytes b with data size (int) before.
func WriteBytes(out io.Writer, b []byte) error {
	err := binary.Write(out, binary.LittleEndian, int64(len(b)))
	if err != nil {
		fmt.Println("Error while writing data size: ", err)
		return err
	}
	_, err = out.Write(b)
	if err != nil {
		fmt.Println("Error while writing data: ", err)
		return err
	}
	return nil
}

const byteBufferSize int64 = 1024

var bytesBuffer = make([]byte, 1024)

// ReadBytes read bytes b with data size (int) before.
func ReadBytes(in io.Reader, b *bytes.Buffer) (bool, error) {
	var dataLen int64
	err := binary.Read(in, binary.LittleEndian, &dataLen)
	if err != nil {
		if err.Error() == "EOF" {
			return true, nil
		}
		fmt.Println("Error while reading data size: ", err)
		return false, err
	}
	var readTotal int64
	for readTotal < dataLen {
		var toRead int64
		if dataLen-readTotal >= byteBufferSize {
			toRead = byteBufferSize
		} else {
			toRead = dataLen - readTotal
		}
		read, err := in.Read(bytesBuffer[:toRead])
		b.Write(bytesBuffer[:toRead])
		readTotal += int64(read)
		if err != nil {
			fmt.Println("Error while reading data: ", err)
			return false, err
		}
	}
	return false, nil
}

// ByteReader is used to read from byte file.
type ByteReader struct {
	isEOF  bool
	input  io.Reader
	buffer bytes.Buffer
}

// HasNext returns true when there is more to read.
func (br *ByteReader) HasNext() bool {
	return br.isEOF
}

// ReadNext tries to read next bytes chunk and returns true when there is more to read.
func (br *ByteReader) ReadNext() bool {
	br.buffer.Reset()
	var err error
	br.isEOF, err = ReadBytes(br.input, &br.buffer)
	if err != nil {
		panic(err)
	}
	return (*br).HasNext()
}

// GetValue return read byte value.
func (br *ByteReader) GetValue() []byte {
	return (*br).buffer.Bytes()
}

// CreateByteReader returns ByteReader.
func CreateByteReader(input io.Reader) *ByteReader {
	result := &ByteReader{
		isEOF: false,
		input: input,
	}
	result.ReadNext()
	return result
}
