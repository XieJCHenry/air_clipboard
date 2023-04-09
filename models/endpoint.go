package models

import (
	"encoding/json"
	"fmt"
)

type EndPoint struct {
	Ip         string `json:"ip"`
	Name       string `json:"name"`
	DeviceName string `json:"deviceName"`
	Key_       string
}

func (ep *EndPoint) JsonString() string {
	bytes, err := json.Marshal(ep)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func (ep *EndPoint) Key() string {
	if ep.Key_ == "" {
		ep.Key_ = fmt.Sprintf("%s", ep.Ip)
	}
	return ep.Key_
}
