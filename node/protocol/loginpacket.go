package protocol

import (
	"bytes"
	"io"
)

type LoginPacket struct {
	Name String
	UUID [16]byte
}

func (h *LoginPacket) ReadFrom(r io.Reader) error {
	var p Packet
	err := p.ReadFrom(r)
	if err != nil {
		return err
	}

	_, err = h.Name.readFrom(p)
	if err != nil {
		return err
	}

	_, err = p.Read(h.UUID[:])
	return err
}

func (h LoginPacket) WriteTo(w io.Writer) error {
	buf := bytes.NewBuffer(make([]byte, 0))

	_, err := h.Name.writeTo(buf)
	if err != nil {
		return err
	}

	_, err = buf.Write(h.UUID[:])
	if err != nil {
		return err
	}

	return Packet{
		ID:   0x00,
		Data: buf,
	}.WriteTo(w)
}
