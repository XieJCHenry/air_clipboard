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
		ep.key = fmt.Sprintf("%s$%s$%s", ep.Ip, ep.Name, ep.DeviceName)
	}
	return ep.key
}
