package packet

import (
	"encoding/json"
	"fmt"

	"github.com/XieJCHenry/gokits/collections/set"
)

type Handler struct {
	supportedProtocols set.Set[ProtocolID]
}

func NewHandler() *Handler {
	return &Handler{
		supportedProtocols: set.New[ProtocolID](),
	}
}

func (h *Handler) Parse(data []byte) (*Packet, error) {
	packet := new(Packet)
	err := json.Unmarshal(data, packet)
	if err != nil {
		return nil, fmt.Errorf("parse packet from binary failed, err = %s", err)
	}
	// 检查packet的协议是否支持
	if !h.isPacketIdValid(packet.Id) {
		return nil, fmt.Errorf("packet id is invalid, id = %v", packet.Id)
	}
	return packet, nil
}

func (h *Handler) isPacketIdValid(id ProtocolID) bool {
	return h.supportedProtocols.Contains(id)
}

func (h *Handler) AddProtocol(id ProtocolID) {
	h.supportedProtocols.InsertIfAbsent(id)
}
