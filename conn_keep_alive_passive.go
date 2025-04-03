package libws

type (
	PingMessageFactory func(content []byte) Message

	PassiveKeepAliveHandler func(ch ConnectionHandler, m Message)
)

// passiveKeepAliveConnectionHandler is a structure that automatically replies to ping/pong messages to keep the connection open.
// It forwards data messages as is.
type passiveKeepAliveConnectionHandler struct {
	// ConnectionHandler is the underlying connection handler
	ConnectionHandler
	// handler is a function that handles passive keep alive messages
	handler PassiveKeepAliveHandler
}

func (h *passiveKeepAliveConnectionHandler) Recv(m Message) {
	h.handler(h.ConnectionHandler, m)

	h.ConnectionHandler.Recv(m)
}

func newPassiveKeepAliveConnectionHandler(
	c ConnectionHandler,
	h PassiveKeepAliveHandler,
) *passiveKeepAliveConnectionHandler {
	return &passiveKeepAliveConnectionHandler{ConnectionHandler: c, handler: h}
}

func NewPassiveKeepAliveConnectionHandlerFactory(
	factory ConnectionHandlerFactory,
	handler PassiveKeepAliveHandler,
) ConnectionHandlerFactory {
	return func(
		client Client,
		msgHandler MessageHandler,
		emitter emitter[EventType, EventType],
	) ConnectionHandler {
		return newPassiveKeepAliveConnectionHandler(factory(client, msgHandler, emitter), handler)
	}
}

func NewPingMessageFactory(pingType MessageType) PingMessageFactory {
	return func(data []byte) Message {
		return NewMessage(pingType, data)
	}
}

func KeepAliveHandlerReplyPingWithPong(ch ConnectionHandler, m Message) {
	if m.Type() == PingMessage {
		ch.Send(NewPongMessage(m.Data()))
	}
}
