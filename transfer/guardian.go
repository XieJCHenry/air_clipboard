package transfer

import (
	"air_clipboard/models"
	"net"
	"time"

	"go.uber.org/zap"
)

type guardian struct {
	logger *zap.SugaredLogger
	id     string
	conn   net.Conn
	exit   chan struct{}
	C      chan *models.Message
}

func newGuardian(logger *zap.SugaredLogger, key string, conn net.Conn) *guardian {
	return &guardian{
		logger: logger,
		id:     key,
		conn:   conn,
		exit:   make(chan struct{}, 1),
		C:      make(chan *models.Message, 1024),
	}
}

func (g *guardian) Id() string {
	return g.id
}

func (g *guardian) Start() {

	// wait until C is clear
	defer func() {
		close(g.C)
		for {
			if len(g.C) == 0 {
				g.logger.Debugf("guardian [%s] truely exit..", g.id)
				break
			} else {
				time.Sleep(5 * time.Second)
			}
		}
	}()

	for {
		select {
		case <-g.exit:
			{
				return
			}
		default:
			{
				buffer := make([]byte, 4096)
				n, err := g.conn.Read(buffer)
				if err != nil {
					g.logger.Errorf("read from failed, err = %s", err)
					continue
				}
				buffer = buffer[:n]
				msg := &models.Message{}
				msg.Unmarshal(buffer)
				g.C <- msg
			}
		}
	}
}

func (g *guardian) Exit() {
	g.exit <- struct{}{}
}
