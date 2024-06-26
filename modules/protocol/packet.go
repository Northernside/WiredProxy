package protocol

import (
	"bytes"
	"encoding/gob"
	"io"
)

/*
	Payload:

	Length Entire Packet (VarInt)
	PacketID (VarInt)
	Data (Byte Array)
*/

func (p *Packet) Write(conn io.Writer) (int64, error) {
	buf := bytes.Buffer{}

	//write packet id to buffer
	_, err := p.ID.WriteTo(&buf)
	if err != nil {
		return 0, err
	}

	//write data to buffer
	_, err = buf.Write(p.Data)
	if err != nil {
		return 0, err
	}

	//write buffer length to connection
	length := VarInt(buf.Len())
	n, err := length.WriteTo(conn)
	if err != nil {
		return n, err
	}

	//write buffer to connection
	return buf.WriteTo(conn)
}

func (p *Packet) Read(conn io.Reader) error {
	// Read packet length
	var length VarInt
	_, err := length.ReadFrom(conn)
	if err != nil {
		return err
	}

	// Create a buffer of the specified length
	buf := make([]byte, length)

	// Read the data into the buffer
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return err
	}

	// Create a bytes buffer from the read data
	dataBuf := bytes.NewBuffer(buf)

	// Read the packet ID from the buffer
	id, err := readVarInt(dataBuf)
	if err != nil {
		return err
	}

	p.ID = id

	// Read the remaining data into p.Data
	p.Data = dataBuf.Bytes()

	return nil
}

func EncodePacket(s any) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(s)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodePacket(data []byte, s any) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(s)
}
