package protocol

import (
	"bytes"
	"io"
)

type HandshakePacket struct {
	PacketId  varInt
	Version   varInt
	Hostname  String
	Port      unsignedShort
	NextState varInt
}

func (h *HandshakePacket) ReadFrom(r io.Reader) error {
	var p Packet
	err := p.ReadFrom(r)
	if err != nil {
		return err
	}

	_, err = h.Version.readFrom(p)
	if err != nil {
		return err
	}

	_, err = h.Hostname.readFrom(p)
	if err != nil {
		return err
	}

	_, err = h.Port.readFrom(p)
	if err != nil {
		return err
	}

	_, err = h.NextState.readFrom(p)
	return err
}

func (h HandshakePacket) WriteTo(w io.Writer) error {
	buf := bytes.NewBuffer(make([]byte, 0))

	_, err := h.Version.writeTo(buf)
	if err != nil {
		return err
	}

	_, err = h.Hostname.writeTo(buf)
	if err != nil {
		return err
	}

	_, err = h.Port.writeTo(buf)
	if err != nil {
		return err
	}

	_, err = h.NextState.writeTo(buf)
	if err != nil {
		return err
	}

	return Packet{
		ID:   0x00,
		Data: buf,
	}.WriteTo(w)
}
