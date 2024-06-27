package protocol

import (
	"io"
	"os"
)

const (
	MTU = 1460
)

type BinaryData struct {
	Label string
	Data  []byte
}

func splitBytes(data []byte, size int) [][]byte {
	var ret [][]byte
	for len(data) > size {
		ret = append(ret, data[:size])
		data = data[size:]
	}

	return append(ret, data)
}

func (c *Conn) SendFile(label, path string, packetIdData VarInt, packetIdEnd VarInt) error {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	for _, packet := range splitBytes(content, MTU-len(label)) {
		err = c.SendPacket(packetIdData, BinaryData{
			Label: label,
			Data:  packet,
		})
		if err != nil {
			return err
		}
	}

	return c.SendPacket(packetIdEnd, BinaryData{
		Label: label,
		Data:  nil,
	})
}
