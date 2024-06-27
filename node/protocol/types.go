package protocol

import (
	"bytes"
	"errors"
	"io"
)

type Packet struct {
	ID   varInt
	Data *bytes.Buffer
}

func (pp Packet) Read(p []byte) (n int, err error) {
	return pp.Data.Read(p)
}

const (
	maxVarIntLen  = 5
	maxVarLongLen = 10
)

type (
	unsignedShort uint16
	String        string
	varInt        int32
)

func (us unsignedShort) writeTo(w io.Writer) (int64, error) {
	n := uint16(us)
	byteLen := uint16(8)
	nn, err := w.Write([]byte{byte(n >> byteLen), byte(n)})
	return int64(nn), err
}

func (us *unsignedShort) readFrom(r io.Reader) (int64, error) {
	var bs [2]byte
	nn, err := io.ReadFull(r, bs[:])
	if err != nil {
		return int64(nn), err
	}

	n := int64(nn)

	*us = unsignedShort(int16(bs[0])<<8 | int16(bs[1]))
	return n, nil
}

func (s String) writeTo(w io.Writer) (int64, error) {
	byteStr := []byte(s)
	n1, err := varInt(len(byteStr)).writeTo(w)
	if err != nil {
		return n1, err
	}

	n2, err := w.Write(byteStr)
	return n1 + int64(n2), err
}

func (v varInt) writeTo(w io.Writer) (int64, error) {
	var vi [maxVarIntLen]byte
	n := v.writeToBytes(vi[:])
	n, err := w.Write(vi[:n])
	return int64(n), err
}

func (v varInt) Len() int {
	switch {
	case v < 0:
		return maxVarIntLen
	case v < 1<<(7*1):
		return 1
	case v < 1<<(7*2):
		return 2
	case v < 1<<(7*3):
		return 3
	case v < 1<<(7*4):
		return 4
	default:
		return 5
	}
}

func (v varInt) writeToBytes(c []byte) int {
	num := uint32(v)
	i := 0
	for {
		b := num & 0x7F
		num >>= 7
		if num != 0 {
			b |= 0x80
		}

		c[i] = byte(b)
		i++
		if num == 0 {
			break
		}
	}

	return i
}

func (s *String) readFrom(r io.Reader) (int64, error) {
	var strLen varInt

	nn, err := strLen.readFrom(r)
	if err != nil {
		return nn, err
	}

	n := nn

	bs := make([]byte, strLen)
	if _, err := io.ReadFull(r, bs); err != nil {
		return n, err
	}

	n += int64(strLen)

	*s = String(bs)
	return n, nil
}

// readByte read one byte from io.Reader
func readByte(r io.Reader) (int64, byte, error) {
	if r, ok := r.(io.ByteReader); ok {
		v, err := r.ReadByte()
		return 1, v, err
	}

	var v [1]byte
	n, err := r.Read(v[:])
	return int64(n), v[0], err
}

func (v *varInt) readFrom(r io.Reader) (int64, error) {
	var vi uint32
	var num, n int64
	for sec := byte(0x80); sec&0x80 != 0; num++ {
		if num > maxVarIntLen {
			return 0, errors.New("varInt is too big")
		}

		var err error
		n, sec, err = readByte(r)
		if err != nil {
			return n, err
		}

		vi |= uint32(sec&0x7F) << uint32(7*num)
	}

	*v = varInt(vi)
	return n, nil
}
