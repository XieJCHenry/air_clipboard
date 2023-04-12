package models

import (
	"encoding/json"
	"fmt"
)

type EndPoint struct {
	Ip         string `json:"ip"`
	Name       string `json:"name"`
	DeviceName string `json:"deviceName"`
	key        string
}

func (ep *EndPoint) JsonString() string {
	bytes, err := json.Marshal(ep)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func (ep *EndPoint) Key() string {
	if ep.key == "" {
		ep.key = fmt.Sprintf("%s", ep.Ip)
	}
	return ep.key
}

// Equal for go-cmp
func (ep *EndPoint) Equal(other *EndPoint) bool {
	return ep.Ip == other.Ip &&
		ep.Name == other.Name &&
		ep.DeviceName == other.DeviceName
}
