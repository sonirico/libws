package libws

import (
	"context"
)

// basicClient is a client implementation with a single connection socket. It only forwards websocket 'data' messages to
// the message messageHandler, whereas 'ping', 'pong' or 'close' messages will be passed down to connection handlers for handling.
// IMPORTANT: Not to be wrapped with subscriber_static client. This client is intended to be use as a standalone client and
// a future replacement for the others.
type basicClient struct {
	// connectionHandlerFactory is a factory for creating new connection handlers
	connectionHandlerFactory ConnectionHandlerFactory
	// connectionHandler is the active connection messageHandler
	connectionHandler ConnectionHandler
	// messageHandler is a messageHandler for processing incoming messages
	messageHandler MessageHandler

	eventHandler func(Client, EventType)

	eventEmitter *EventEmitterCallback[EventType, EventType]
}

func (b *basicClient) createConnectionHandler(_ context.Context) {
	handlerWrapper := func(cli Client, m Message) {
		if m.Type().IsData() {
			b.messageHandler(cli, m)
		} else {
			b.connectionHandler.Recv(m)
		}
	}

	b.connectionHandler = b.connectionHandlerFactory(b, handlerWrapper, b.eventEmitter)
}

func (b *basicClient) Open(ctx context.Context) error {
	b.createConnectionHandler(ctx)

	b.eventEmitter.On(EventConnect, func(eventType EventType) {
		b.eventHandler(b, eventType)
	})

	b.eventEmitter.On(EventClose, func(eventType EventType) {
		b.eventHandler(b, eventType)
	})

	b.eventEmitter.On(EventReconnect, func(eventType EventType) {
		b.eventHandler(b, eventType)
	})

	if err := b.connectionHandler.Connect(ctx); err != nil {
		return err
	}

	return nil
}

func (b *basicClient) Send(m Message) {
	b.connectionHandler.Send(m)
}

func (b *basicClient) Close() {
	if b.eventEmitter != nil {
		b.eventEmitter.Close()
	}
	if b.connectionHandler != nil {
		b.connectionHandler.Close()
	}
}

func (b *basicClient) CloseChan() CloseChan {
	return b.connectionHandler.CloseChan()
}

func newBasicClient(
	connHandlerFactory ConnectionHandlerFactory,
	messageHandler MessageHandler,
	eventHandler EventHandler,
) *basicClient {
	return &basicClient{
		messageHandler:           messageHandler,
		eventHandler:             eventHandler,
		connectionHandlerFactory: connHandlerFactory,
		eventEmitter:             NewEventEmitter[EventType, EventType](),
	}
}

func NewBasicClientFactory(
	connHandlerFactory ConnectionHandlerFactory,
	messageHandler MessageHandler,
	eventHandler EventHandler,
) ClientFactory {
	return func() Client {
		return newBasicClient(
			connHandlerFactory,
			messageHandler,
			eventHandler,
		)
	}
}
