package transfer

import (
	"air_clipboard/models"
	"net"

	"go.uber.org/zap"
)

type guardian struct {
	logger     *zap.SugaredLogger
	id         string
	conn       net.Conn
	exit       chan struct{}
	sendChan   chan *models.Message
	submitChan chan *models.Message
}

func newGuardian(logger *zap.SugaredLogger, key string, conn net.Conn, submitChan chan *models.Message) *guardian {
	return &guardian{
		logger:     logger,
		id:         key,
		conn:       conn,
		exit:       make(chan struct{}, 1),
		sendChan:   make(chan *models.Message, 32),
		submitChan: submitChan,
	}
}

func (g *guardian) Id() string {
	return g.id
}

func (g *guardian) Start() {

	for {
		select {
		case <-g.exit:
			{
				return
			}
		case msg := <-g.sendChan:
			{
				_, err := g.conn.Write(msg.Marshal())
				if err != nil {
					g.logger.Errorf("write failed, err = %s", err)
					continue
				}
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
				g.submitChan <- msg
			}
		}
	}
}

func (g *guardian) Send(msg *models.Message) {
	if msg != nil {
		g.sendChan <- msg
	}
}

func (g *guardian) Exit() {
	g.exit <- struct{}{}
}
