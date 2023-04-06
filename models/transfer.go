package models

import "encoding/json"

type Message struct {
	Header *MessageHeader `json:"header"`
	Body   *MessageBody   `json:"body"`
}

type MessageHeader struct {
}

type MessageBody struct {
	Content string `json:"content"`
}

func (m *Message) Marshal() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

func (m *Message) Unmarshal(bytes []byte) {
	temp := &Message{}
	err := json.Unmarshal(bytes, temp)
	if err != nil {
		return
	}
	m.Header = temp.Header
	m.Body = temp.Body
}
