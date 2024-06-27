package protocol

import (
	"bytes"
	"io"
)

type StatusResponse struct {
	Status String
}

type (
	StatusResponseJSON struct {
		Version     Version     `json:"version"`
		Players     Players     `json:"players"`
		Description Description `json:"description"`
		Favicon     string      `json:"favicon"`
		Modinfo     Modinfo     `json:"modinfo"`
	}
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	}
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	}
	Modinfo struct { // forge
		Type    string        `json:"type"`
		ModList []interface{} `json:"modList"`
	}
	Description struct {
		Extra `json:"extra"`
		Text  string `json:"text"`
	}
	Extra []struct {
		Color string `json:"color"`
		Text  string `json:"text"`
		Bold  bool   `json:"bold,omitempty"`
	}
)

func (s *StatusResponse) ReadFrom(r io.Reader) error {
	var p Packet
	err := p.ReadFrom(r)
	if err != nil {
		return err
	}

	_, err = s.Status.readFrom(p)
	return err
}

func (s StatusResponse) WriteTo(w io.Writer) error {
	buf := bytes.NewBuffer(make([]byte, 0))

	_, err := s.Status.writeTo(buf)
	if err != nil {
		return err
	}
	return Packet{
		ID:   0x00,
		Data: buf,
	}.WriteTo(w)
}
