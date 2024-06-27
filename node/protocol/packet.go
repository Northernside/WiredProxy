package protocol

import (
	"bytes"
	"io"
)

func (p *Packet) ReadFrom(r io.Reader) error {
	var packetLength varInt
	_, err := packetLength.readFrom(r)
	if err != nil {
		return err
	}

	var packet_id varInt
	l2, err := packet_id.readFrom(r)
	if err != nil {
		return err
	}

	p.ID = packet_id
	b := make([]byte, int(packetLength)-int(l2))
	_, err = io.ReadFull(r, b)
	if err != nil {
		return err
	}

	p.Data = bytes.NewBuffer(b)
	return nil
}

func (p Packet) WriteTo(w io.Writer) error { //TODO
	buflen := 0
	if p.Data != nil {
		buflen = p.Data.Len()
	}
	_, err := varInt(p.ID.Len() + buflen).writeTo(w)
	if err != nil {
		return err
	}
	_, err = p.ID.writeTo(w)
	if err != nil {
		return err
	}
	if p.Data != nil {

		_, err = p.Data.WriteTo(w)
	}
	return err
}
