package packet

import (
	"encoding/json"
)

type ProtocolID int

type Packet struct {
	Id   ProtocolID `json:"id,omitempty"`
	Body []byte     `json:"body,omitempty"`
}

func NewPacket(id ProtocolID, body []byte) *Packet {
	return &Packet{
		Id:   id,
		Body: body,
	}
}

func Marshal(packet *Packet) ([]byte, error) {
	return json.Marshal(packet)
}

func Unmarshal(data []byte) (*Packet, error) {
	packet := new(Packet)
	err := json.Unmarshal(data, packet)
	return packet, err
}

func (p *Packet) GetBody() []byte {
	return p.Body
}
