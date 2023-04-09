package discovery

import "air_clipboard/models"

type EndPointStatus string

const (
	StatusOnline  EndPointStatus = "online"
	StatusOffline EndPointStatus = "offline"
)

type EndpointPacket struct {
	From   *models.EndPoint `json:"from"`
	Status EndPointStatus   `json:"status"`
}

type DiscoverEventType int

const (
	EventUnknown = iota
	EventAddEndPoint
	EventDeleteEventPoint
)

type DiscoveryEvent struct {
	Type     DiscoverEventType
	Endpoint *models.EndPoint
}
