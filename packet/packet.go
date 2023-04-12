package packet

import (
	"bytes"
	"encoding/binary"
)

type ProtocolID int

type Packet struct {
	Id   ProtocolID
	body []byte
}

func NewPacket(id ProtocolID, body []byte) *Packet {
	return &Packet{
		Id:   id,
		body: body,
	}
}

func Marshal(packet *Packet) ([]byte, error) {
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.LittleEndian, *packet)
	return buffer.Bytes(), err
}

func Unmarshal(data []byte) (*Packet, error) {
	buffer := bytes.NewBuffer(data)
	packet := new(Packet)
	err := binary.Read(buffer, binary.LittleEndian, packet)
	return packet, err
}

func (p *Packet) Body() []byte {
	return p.body
}
