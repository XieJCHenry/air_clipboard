package transfer

import (
	"air_clipboard/models"
	"fmt"
	"net"
	"sync"

	"github.com/XieJCHenry/gokits/collections/gmap"
	"go.uber.org/zap"
)

type Postman interface {
	AddTransfer(target *models.EndPoint)
	RemoveTransfer(target *models.EndPoint)
}

// todo 需要从guardian 中获取Channel来读取信息
type postman struct {
	logger       *zap.SugaredLogger
	transferPort int
	mtx          sync.Mutex
	connPool     gmap.Map[string, net.Conn]
	guardians    gmap.Map[string, *guardian]
}

func New(logger *zap.SugaredLogger, port int) Postman {
	return &postman{
		logger:       logger,
		transferPort: port,
		mtx:          sync.Mutex{},
		connPool:     gmap.New[string, net.Conn](),
		guardians:    gmap.New[string, *guardian](),
	}
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
	g := newGuardian(p.logger, target.Key(), conn)
	go g.Start()

	p.guardians.PutIfAbsent(g.Id(), g)
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
