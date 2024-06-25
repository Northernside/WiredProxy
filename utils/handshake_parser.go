package handshake

import (
	"bytes"
	"errors"
	"io"
)

type (
	unsignedShort uint16
	String        string
	varInt        int32
)

type HandshakePacket struct {
	PacketId  varInt
	Version   varInt
	Hostname  String
	Port      unsignedShort
	NextState varInt
}

func ReadPacket(c io.Reader) (*HandshakePacket, error) {
	var packetLength varInt
	_, err := packetLength.readFrom(c)
	if err != nil {
		return nil, err
	}

	var packet_id varInt
	l2, err := packet_id.readFrom(c)
	if err != nil {
		return nil, err
	}

	b := make([]byte, int(packetLength)-int(l2))
	_, err = io.ReadFull(c, b)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(b)

	s := HandshakePacket{}
	s.PacketId = packet_id
	_, err = s.Version.readFrom(buf)
	if err != nil {
		return nil, err
	}

	_, err = s.Hostname.readFrom(buf)
	if err != nil {
		return nil, err
	}

	_, err = s.Port.readFrom(buf)
	if err != nil {
		return nil, err
	}

	_, err = s.NextState.readFrom(buf)
	return &s, err
}

func WritePacket(packet HandshakePacket, c io.Writer) error {
	buf := bytes.NewBuffer(make([]byte, 0))

	_, err := varInt(packet.PacketId).writeTo(buf)

	_, err = packet.Version.writeTo(buf)
	if err != nil {
		return err
	}

	_, err = packet.Hostname.writeTo(buf)
	if err != nil {
		return err
	}

	_, err = packet.Port.writeTo(buf)
	if err != nil {
		return err
	}

	_, err = packet.NextState.writeTo(buf)
	if err != nil {
		return err
	}

	len := varInt(buf.Len())
	_, err = len.writeTo(c)
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(c)

	return err
}

const (
	maxVarIntLen  = 5
	maxVarLongLen = 10
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

// readByte read one byte from io.Reader.
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
