package discovery

import (
	"air_clipboard/models"
	"air_clipboard/packet"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/XieJCHenry/gokits/collections/slice"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
)

const (
	DefaultPort              = 9456
	DefaultDiscoveryInterval = 1 // min
)

const (
	Login        packet.ProtocolID = 1
	Logout       packet.ProtocolID = 2
	GetAllOnline packet.ProtocolID = 3
)

type EndPointDiscovery interface {
	Start()
	EndPoints() slice.Slice[*models.EndPoint]
	GetSelfInfo() *models.EndPoint
	BroadcastSelf()
	Stop()
	OnDiscoverEvent() chan *DiscoveryEvent
}

type endPointDiscovery struct {
	udpPort          int
	interval         int
	udpConn          *net.UDPConn
	selfInfo         *models.EndPoint
	endpoints        slice.Slice[*models.EndPoint]
	localAddrs       slice.Slice[string]
	mtx              sync.Mutex
	stopChan         chan struct{}
	discoverEvenChan chan *DiscoveryEvent
	logger           *zap.SugaredLogger
	packetHandler    *packet.Handler
}

func New(logger *zap.SugaredLogger, udpPort int, discoveryInterval int, selfInfo *models.EndPoint) EndPointDiscovery {
	if udpPort <= 0 {
		udpPort = DefaultPort
	}
	if discoveryInterval <= 0 {
		discoveryInterval = DefaultDiscoveryInterval
	}

	packetHandler := packet.NewHandler()
	packetHandler.AddProtocol(Login)
	packetHandler.AddProtocol(Logout)
	packetHandler.AddProtocol(GetAllOnline)
	return &endPointDiscovery{
		udpPort:          udpPort,
		interval:         discoveryInterval,
		endpoints:        slice.New[*models.EndPoint](),
		localAddrs:       slice.New[string](),
		mtx:              sync.Mutex{},
		stopChan:         make(chan struct{}, 1),
		discoverEvenChan: make(chan *DiscoveryEvent),
		logger:           logger,
		selfInfo:         selfInfo,
		packetHandler:    packetHandler,
	}
}

func (e *endPointDiscovery) Start() {
	ticker := time.NewTicker(time.Duration(e.interval) * time.Minute)
	defer ticker.Stop()

	e.initLocalNetAddr()

	go e.startReceiver()
	time.Sleep(5)
	e.broadcastSelfInfo()

	for {
		select {
		case <-ticker.C:
			e.broadcastSelfInfo()
		case <-e.stopChan:
			break
		}
	}
}

func (e *endPointDiscovery) startReceiver() {
	con, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: e.udpPort,
	})
	if err != nil {
		e.logger.Fatalf("listen udp %d failed, err=%s", e.udpPort, err)
	}
	e.udpConn = con
	e.logger.Info("start ...")
	defer e.udpConn.Close()

	for {
		select {
		case <-e.stopChan:
			{
				break
			}
		default:
			{
				var data [1024]byte
				n, addr, err := e.udpConn.ReadFromUDP(data[:])
				if err != nil {
					e.logger.Errorf("read from udp failed, err = %s", err)
					continue
				}
				// 获取收到的数据包，解析是否是air_clipboard其他endpoint发来的
				pack, err := e.packetHandler.Parse(data[:n])
				if err != nil {
					e.logger.Errorf("parse packet failed, err = %s", err)
					continue
				}
				if err = e.handlePacket(pack, addr); err != nil {
					e.logger.Errorf("handle packet failed, err = %s", err)
				}
			}
		}
	}
}

func (e *endPointDiscovery) broadcastSelfInfo() {
	remoteAddr := &net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: e.udpPort,
	}
	//e.udpConn.Write()
	remoteCon, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		e.logger.Fatalf("listen udp %d failed, err=%s", e.udpPort, err)
	}

	sendBytes, err := e.marshalSelfInfo()
	if err != nil {
		e.logger.Errorf("marshalSelfInfo failed, err=%s", err)
	}
	_, err = remoteCon.Write(sendBytes)
	if err != nil {
		e.logger.Errorf("client write failed, err=%s", err)
		return
	}
	e.logger.Info("broadcast self to remote...")
}

func (e *endPointDiscovery) uniCastSelfInfo(remoteAddr *net.UDPAddr) {
	sendBytes, err := e.marshalSelfInfo()
	if err != nil {
		return
	}

	_, err = e.udpConn.WriteToUDP(sendBytes, remoteAddr)
	if err != nil {
		e.logger.Errorf("client write failed, err=%s", err)
	}
}

func (e *endPointDiscovery) EndPoints() slice.Slice[*models.EndPoint] {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	endpoints := slice.New[*models.EndPoint]()
	for i := 0; i < e.endpoints.Size(); i++ {
		endpoints.Append(e.endpoints.At(i))
	}
	return endpoints
}

func (e *endPointDiscovery) GetSelfInfo() *models.EndPoint {
	return e.selfInfo
}

func (e *endPointDiscovery) BroadcastSelf() {
	e.broadcastSelfInfo()
}

func (e *endPointDiscovery) Stop() {
	e.stopChan <- struct{}{}
}

func (e *endPointDiscovery) OnDiscoverEvent() chan *DiscoveryEvent {
	return e.discoverEvenChan
}

func (e *endPointDiscovery) handlePacket(pack *packet.Packet, addr *net.UDPAddr) error {
	endpointPacket := &EndpointPacket{}
	err := json.Unmarshal(pack.Body(), endpointPacket)
	if err != nil {
		e.logger.Errorf("unmarshal packet from other failed, err=%s", err)
		return err
	}
	endpointPacket.From.Ip = addr.IP.String()
	switch pack.Id {
	case Login:
		{
			// 添加这个节点，并且将自身的节点信息告诉它
			e.updateCache(endpointPacket)
			e.uniCastSelfInfo(addr)
		}
	case Logout:
		{
			// 删除这个节点
			e.updateCache(endpointPacket)
		}
	}
	return nil
}

func (e *endPointDiscovery) updateCache(packet *EndpointPacket) {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	endPoint := packet.From
	if e.localAddrs.Contains(endPoint.Ip) || cmp.Equal(endPoint, e.selfInfo) {
		e.logger.Infof("skip local connect")
		return
	}
	e.logger.Infof("receive packet from %s", endPoint.Ip)

	if packet.Status == StatusOnline {
		e.endpoints.AppendIfAbsent(endPoint)
		e.discoverEvenChan <- &DiscoveryEvent{
			Type:     EventAddEndPoint,
			Endpoint: endPoint,
		}
	} else if packet.Status == StatusOffline {
		e.endpoints.RemoveIfPresent(endPoint)
		e.discoverEvenChan <- &DiscoveryEvent{
			Type:     EventDeleteEventPoint,
			Endpoint: endPoint,
		}
	} else {
		e.logger.Errorf("unknown status '%s' from endpoint = %v", packet.Status, endPoint)
	}
}

func (e *endPointDiscovery) initLocalNetAddr() {
	interfaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(fmt.Sprintf("init net interface addrs failed, err = %s", err))
	}

	for _, address := range interfaceAddrs {
		ipNet, isVailIpNet := address.(*net.IPNet)
		if isVailIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				e.localAddrs.Append(ipNet.IP.To4().String())
			}
		}
	}
}

func (e *endPointDiscovery) marshalSelfInfo() ([]byte, error) {
	bytes, err := json.Marshal(&EndpointPacket{
		From:   e.selfInfo,
		Status: StatusOnline,
	})
	if err != nil {
		return nil, err
	}
	pack := packet.NewPacket(GetAllOnline, bytes)
	sendBytes, err := packet.Marshal(pack)
	if err != nil {
		return nil, err
	}
	return sendBytes, nil
}
