package transfer

import (
	"air_clipboard/models"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/XieJCHenry/gokits/collections/gmap"
	"go.uber.org/zap"
)

type Postman interface {
	Start()
	AddTransfer(target *models.EndPoint)
	RemoveTransfer(target *models.EndPoint)
	TransferTo(endPointKey string, message *models.Message) error
	Broadcast(content string)
	RecvFrom() chan *models.Message
	GetSelfInfo() *models.EndPoint
}

// todo 需要从guardian 中获取Channel来读取信息
type postman struct {
	logger       *zap.SugaredLogger
	transferPort int
	mtx          sync.Mutex
	connPool     gmap.Map[string, net.Conn]
	guardians    gmap.Map[string, *guardian]
	selfInfo     *models.EndPoint
	recvChan     chan *models.Message
}

func New(logger *zap.SugaredLogger, port int, selfInfo *models.EndPoint) Postman {
	return &postman{
		logger:       logger,
		transferPort: port,
		mtx:          sync.Mutex{},
		connPool:     gmap.New[string, net.Conn](),
		guardians:    gmap.New[string, *guardian](),
		selfInfo:     selfInfo,
		recvChan:     make(chan *models.Message, 1024),
	}
}

func (p *postman) Start() {

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", p.transferPort))
	if err != nil {
		panic(fmt.Sprintf("resolve tcp addr failed, err = %s", err))
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("postman listen at %d failed, err = %s", p.transferPort, err))
	}
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			p.logger.Errorf("accept tcp failed, err = %s", err)
			continue
		}
		remoteAddr := conn.RemoteAddr().String()
		elems := strings.SplitN(remoteAddr, ":", 2)
		if len(elems) != 2 {
			continue
		}
		remoteIp, _ := elems[0], elems[1]
		newEndpoint := &models.EndPoint{Ip: remoteIp}
		p.AddTransfer(newEndpoint)
	}
}

func (p *postman) AddTransfer(target *models.EndPoint) {
	var (
		conn net.Conn
		err  error
	)

	conn, err = p.getOrInitConn(target)
	if err != nil {
		p.logger.Errorf("add transfer[%s] failed, cann't get or init connection, err = %s", target, err)
		return
	}
	g := newGuardian(p.logger, target.Key(), conn, p.recvChan)
	go g.Start()

	p.guardians.PutIfAbsent(target.Key(), g)
}

func (p *postman) RemoveTransfer(target *models.EndPoint) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if g, ok := p.guardians.DeleteIfPresent(target.Key()); ok {
		g.Exit()
		if conn, ok := p.connPool.DeleteIfPresent(target.Key()); ok {
			conn.Close()
			p.logger.Infof("remove transfer %s", target)
		}
	}
}

func (p *postman) TransferTo(endPointKey string, message *models.Message) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if message.Header == nil {
		message.Header = &models.MessageHeader{
			Sender:   fmt.Sprintf("%s-%s", p.selfInfo.Name, p.selfInfo.DeviceName),
			SendTime: time.Now().Unix(),
		}
	}

	g := p.guardians.Get(endPointKey)
	if g != nil {
		g.Send(message)
	}
	return nil
}

func (p *postman) Broadcast(content string) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	msg := &models.Message{
		Header: &models.MessageHeader{
			Sender:   fmt.Sprintf("%s-%s", p.selfInfo.Name, p.selfInfo.DeviceName),
			SendTime: time.Now().Unix(),
		},
		Body: &models.MessageBody{
			Content: content,
		},
	}

	guardians := p.guardians.Values()
	for i := range guardians {
		g := guardians[i]
		g.Send(msg)
	}
}

func (p *postman) RecvFrom() chan *models.Message {
	return p.recvChan
}

func (p *postman) GetSelfInfo() *models.EndPoint {
	return p.selfInfo
}

func (p *postman) getOrInitConn(endpoint *models.EndPoint) (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := endpoint.Key()
	addr := fmt.Sprintf("%s:%d", endpoint.Ip, p.transferPort)
	if conn = p.connPool.GetOrDefault(key, nil); conn == nil {
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			p.logger.Errorf("dial remote[%s] failed, err = %s", key, err)
			return nil, err
		}
		p.connPool.Put(key, conn)
	}
	return conn, nil
}
