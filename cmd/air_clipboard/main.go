package main

import (
	"air_clipboard/discovery"
	"log"
	"net"
)

var logger = log.Default()

const (
	RecvPort = 9456
	SendPort = 9457
)

func main() {

	discoveryService := discovery.New(SendPort, 5)
	go discoveryService.Start()

}

func broadcastAddressList() []*net.IPNet {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Fatalf("addrs failed, err=%s", err)
	}

	var (
		result []*net.IPNet
	)
	for i, addr := range addrs {
		ip, ok := addr.(*net.IPNet)
		if ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				it, _ := net.InterfaceByIndex(i)
				logger.Printf("ip: %s, mask: %s, mac: %v", ip.IP, ip.Mask, it)
				result = append(result, ip)
			}
		}
	}
	return result
}
