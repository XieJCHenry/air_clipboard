package main

import (
	"air_clipboard/discovery"
	"air_clipboard/transfer"
)

type Clipboard struct {
	discovery discovery.EndPointDiscovery
	transfer  transfer.Postman
}
