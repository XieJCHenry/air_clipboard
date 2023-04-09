package discovery

import (
	"air_clipboard/models"
	"encoding/json"
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

/**
EndPoint
局域网内每个设备都是一个endpoint

todo 新增数据包协议类型，功能：启动时广播自身的信息，接收到的节点应立即回复它的信息
*/

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
	selfInfo         *models.EndPoint
	endpoints        slice.Slice[*models.EndPoint]
	mtx              sync.Mutex
	stopChan         chan struct{}
	discoverEvenChan chan *DiscoveryEvent
	logger           *zap.SugaredLogger
}

func New(logger *zap.SugaredLogger, udpPort int, discoveryInterval int, selfInfo *models.EndPoint) EndPointDiscovery {
	if udpPort <= 0 {
		udpPort = DefaultPort
	}
	if discoveryInterval <= 0 {
		discoveryInterval = DefaultDiscoveryInterval
	}
	return &endPointDiscovery{
		udpPort:          udpPort,
		interval:         discoveryInterval,
		endpoints:        slice.New[*models.EndPoint](),
		mtx:              sync.Mutex{},
		stopChan:         make(chan struct{}, 1),
		discoverEvenChan: make(chan *DiscoveryEvent),
		logger:           logger,
		selfInfo:         selfInfo,
	}
}

func (e *endPointDiscovery) Start() {
	ticker := time.NewTicker(time.Duration(e.interval) * time.Minute)
	defer ticker.Stop()

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
	e.logger.Info("start ...")
	defer con.Close()

	for {
		select {
		case <-e.stopChan:
			{
				break
			}
		default:
			{
				var data [1024]byte
				n, addr, err := con.ReadFromUDP(data[:])
				if err != nil {
					e.logger.Errorf("read from udp failed, err=%s", err)
					continue
				}
				// 获取收到的数据包，解析是否是air_clipboard其他endpoint发来的

				if packet, ok := e.parsePacket(data[:n]); ok {
					packet.From.Ip = addr.IP.String()
					e.updateCache(packet)
				}
			}
		}
	}
}

func (e *endPointDiscovery) broadcastSelfInfo() {
	// todo 获取本地在局域网的ip
	localAddr := &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 0,
	}
	remoteAddr := &net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: e.udpPort,
	}
	remoteCon, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		e.logger.Fatalf("listen udp %d failed, err=%s", e.udpPort, err)
	}

	if packetBytes, ok := e.preparePacket(); ok {
		_, err = remoteCon.Write(packetBytes)
		if err != nil {
			e.logger.Errorf("client write failed, err=%s", err)
			return
		}
		e.logger.Info("write to remote...")
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

func (e *endPointDiscovery) preparePacket() ([]byte, bool) {
	packet := &EndpointPacket{
		From:   e.selfInfo,
		Status: StatusOnline,
	}
	bytes, err := json.Marshal(packet)
	return bytes, err == nil
}

func (e *endPointDiscovery) parsePacket(bytes []byte) (*EndpointPacket, bool) {
	packet := &EndpointPacket{}
	err := json.Unmarshal(bytes, packet)
	if err != nil {
		e.logger.Errorf("unmarshal packet from other failed, err=%s", err)
		return nil, false
	}
	return packet, true
}

func (e *endPointDiscovery) updateCache(packet *EndpointPacket) {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	endPoint := packet.From
	if cmp.Equal(endPoint, e.selfInfo) {
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
