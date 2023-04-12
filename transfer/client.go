package transfer

import (
	"air_clipboard/discovery"
	"air_clipboard/models"
	"air_clipboard/models/message"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Postman interface {
	Start()
	TransferTo(endpoint *models.EndPoint, message *message.Message) error
	Broadcast(content string)
	RecvFrom() chan *message.Message
	GetSelfInfo() *models.EndPoint
}

type postman struct {
	logger       *zap.SugaredLogger
	transferPort int
	mtx          sync.Mutex
	lAddr        *net.TCPAddr
	selfInfo     *models.EndPoint
	recvChan     chan *message.Message
	discovery    discovery.EndPointDiscovery
}

func New(logger *zap.SugaredLogger, port int, selfInfo *models.EndPoint, discovery discovery.EndPointDiscovery) Postman {
	return &postman{
		logger:       logger,
		transferPort: port,
		mtx:          sync.Mutex{},
		selfInfo:     selfInfo,
		recvChan:     make(chan *message.Message, 1024),
		discovery:    discovery,
	}
}

func (p *postman) Start() {

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", p.transferPort))
	if err != nil {
		panic(fmt.Sprintf("resolve tcp addr failed, err = %s", err))
	}
	p.lAddr = addr
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("postman listen at %d failed, err = %s", p.transferPort, err))
	}
	p.logger.Infof("postman listen at %s", addr)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			p.logger.Errorf("accept tcp failed, err = %s", err)
			continue
		}
		go func(conn *net.TCPConn) {
			defer conn.Close()

			bytes, err := io.ReadAll(conn)
			if err != nil {
				return
			}
			msg := &message.Message{}
			msg.Unmarshal(bytes)
			p.recvChan <- msg
		}(conn)
	}
}

func (p *postman) TransferTo(endpoint *models.EndPoint, msg *message.Message) error {
	if msg.Header == nil {
		msg.Header = &message.Header{
			Sender:   fmt.Sprintf("%s-%s", p.selfInfo.Name, p.selfInfo.DeviceName),
			SendTime: time.Now().Unix(),
		}
	}

	rAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", endpoint.Ip, p.transferPort))
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", p.lAddr, rAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(msg.Marshal())
	if err != nil {
		return err
	}

	return nil
}

func (p *postman) Broadcast(content string) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	msg := &message.Message{
		Header: &message.Header{
			Sender:   fmt.Sprintf("%s-%s", p.selfInfo.Name, p.selfInfo.DeviceName),
			SendTime: time.Now().Unix(),
		},
		Body: &message.Body{
			Content: content,
		},
	}

	endpoints := p.discovery.EndPoints().ToBuiltIn()
	for _, point := range endpoints {
		err := p.TransferTo(point, msg)
		if err != nil {
			p.logger.Errorf("transfer to %v failed, err = %s", point, err)
		}
	}
}

func (p *postman) RecvFrom() chan *message.Message {
	return p.recvChan
}

func (p *postman) GetSelfInfo() *models.EndPoint {
	return p.selfInfo
}
