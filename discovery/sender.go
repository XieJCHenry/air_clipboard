package discovery

import (
	"log"
	"net"
)

type sender struct {
	udpPort  int
	logger   *log.Logger
	stopChan *chan struct{}
	dataChan chan []byte
}

func (s *sender) Start() {
	con, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: s.udpPort,
	})
	if err != nil {
		s.logger.Fatalf("listen udp %d failed, err=%s", s.udpPort, err)
	}
	s.logger.Printf("start %s ...")

	for {
		select {
		case <-*s.stopChan:
			{
				break
			}
		case bytes := <-s.dataChan:
			{
				con.Write(bytes)
			}
		default:
			{
				var data [1024]byte
				n, addr, err := con.ReadFromUDP(data[:])
				if err != nil {
					s.logger.Printf("read from udp failed, err=%s", err)
					continue
				}
				s.logger.Printf("read from udp %s, data = %s", addr, string(data[:n]))
			}
		}
	}
}

func (s *sender) Send(bytes []byte) {
	s.dataChan <- bytes
}
