package libws

import "fmt"

type MessageType byte

const (
	PingMessage   MessageType = 9
	PongMessage   MessageType = 10
	BinaryMessage MessageType = 2
	DataMessage   MessageType = 1
	CloseError    MessageType = 8
)

func (t MessageType) Is(other MessageType) bool {
	return t == other
}

func (t MessageType) IsData() bool {
	return t.Is(DataMessage)
}

func (t MessageType) IsPing() bool {
	return t.Is(PingMessage)
}

func (t MessageType) IsPong() bool {
	return t.Is(PongMessage)
}

func (t MessageType) IsClose() bool {
	return t.Is(CloseError)
}

type Message interface {
	Type() MessageType
	Data() []byte
	String() string
}

type ErrorMessage interface {
	Message
	Error() string
}

type message struct {
	MessageType MessageType
	MessageData []byte
}

func (m message) Type() MessageType {
	return m.MessageType
}

func (m message) Data() []byte {
	return m.MessageData
}

func (m message) String() string {
	return fmt.Sprintf("Message{type=%d,data=%s}",
		m.MessageType, m.MessageData)
}

type closeMessage struct {
	message
	Code int
}

func (m closeMessage) String() string {
	return fmt.Sprintf("Message{type=%d,code=%d,data=%s}",
		m.message.Type(), m.Code, m.message.Data())
}

func (m closeMessage) Error() string {
	return m.String()
}

func NewMessage(mt MessageType, data []byte) Message {
	return message{MessageType: mt, MessageData: data}
}

func NewDataMessage(data []byte) Message {
	return NewMessage(DataMessage, data)
}

func NewBinaryMessage(data []byte) Message {
	return NewMessage(BinaryMessage, data)
}

func NewPingMessage(data []byte) Message {
	return NewMessage(PingMessage, data)
}

func NewPongMessage(data []byte) Message {
	return NewMessage(PongMessage, data)
}

func NewCloseMessage(code int, data []byte) ErrorMessage {
	return closeMessage{
		message: message{MessageType: CloseError, MessageData: data},
		Code:    code,
	}
}
