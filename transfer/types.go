package transfer

type ContentType int
type Sender string

const (
	ContentTypeUnknown = iota
	ContentTypeText
)

type BaseMessage struct {
	Type     ContentType
	Content  []byte
	SendTime int64
	RecvTime int64
	Sender   Sender
}

func (bm *BaseMessage) GetSender() Sender {
	return bm.Sender
}

func (bm *BaseMessage) GetSendTime() int64 {
	return bm.SendTime
}

func (bm *BaseMessage) GetRecvTime() int64 {
	return bm.RecvTime
}

func (bm *BaseMessage) GetType() ContentType {
	return bm.Type
}
