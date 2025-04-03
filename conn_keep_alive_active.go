package libws

import (
	"context"
	"sync"
	"time"
)

type KeepAliveMessageFactory func() Message

// activeKeepAliveConnectionHandler is a type of ConnectionHandler that automatically sends
// periodic ping messages to keep the connection alive.
// It embeds the ConnectionHandler interface to inherit its methods.
type activeKeepAliveConnectionHandler struct {
	ConnectionHandler
	pingInterval            time.Duration
	keepAliveMessageFactory KeepAliveMessageFactory
	logger                  logger

	connectOnce sync.Once
	closeOnce   sync.Once
	closeC      chan struct{}
}

// Connect sets up the connection and starts the routine for sending periodic keep-alive messages.
// This method is blocking and returns when the connection is no longer alive.
// It only executes once, subsequent calls have no effect.
func (h *activeKeepAliveConnectionHandler) Connect(ctx context.Context) (err error) {
	h.connectOnce.Do(func() {
		err = h.ConnectionHandler.Connect(ctx)

		go h.run(ctx)
	})

	return
}

// Close terminates the connection and stops the keep-alive routine.
// It only executes once, subsequent calls have no effect.
func (h *activeKeepAliveConnectionHandler) Close() {
	h.closeOnce.Do(func() {
		h.ConnectionHandler.Close()
		close(h.closeC)
	})
}

// run initiates the routine that sends keep-alive messages at regular intervals defined by pingInterval.
// It stops when the context is done or the connection is closed.
func (h *activeKeepAliveConnectionHandler) run(ctx context.Context) {
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.ConnectionHandler.Send(h.keepAliveMessageFactory())
		case <-h.closeC:
			return
		}
	}
}

// newActiveKeepAliveConnectionHandler initializes and returns a new ActiveKeepAliveConnectionHandler.
// It takes a ConnectionHandler, a time.Duration, and a KeepAliveMessageFactory as parameters.
// The time.Duration parameter defines the interval between each keep-alive message.
// The KeepAliveMessageFactory generates the keep-alive message to be sent.
func newActiveKeepAliveConnectionHandler(
	logger logger,
	ch ConnectionHandler,
	interval time.Duration,
	keepAliveMessageFactory KeepAliveMessageFactory,
) *activeKeepAliveConnectionHandler {
	return &activeKeepAliveConnectionHandler{
		ConnectionHandler:       ch,
		logger:                  logger,
		pingInterval:            interval,
		keepAliveMessageFactory: keepAliveMessageFactory,
		closeC:                  make(chan struct{}),
	}
}

// NewActiveKeepAliveConnectionHandlerFactory returns a factory function for creating ActiveKeepAliveConnectionHandlers.
// It takes a ConnectionHandlerFactory, a time.Duration, and a KeepAliveMessageFactory as parameters.
// The ConnectionHandlerFactory is used to create the underlying ConnectionHandler.
// The time.Duration parameter sets the interval between each keep-alive message.
// The KeepAliveMessageFactory generates the keep-alive message to be sent.
func NewActiveKeepAliveConnectionHandlerFactory(
	logger logger,
	factory ConnectionHandlerFactory,
	interval time.Duration,
	keepAliveMessageFactory KeepAliveMessageFactory,
) ConnectionHandlerFactory {
	return func(client Client, handler MessageHandler, emitter emitter[EventType, EventType]) ConnectionHandler {
		return newActiveKeepAliveConnectionHandler(
			logger.WithField("subtype", "activeKeepAliveConnectionHandler"),
			factory(client, handler, emitter),
			interval,
			keepAliveMessageFactory,
		)
	}
}

// NewKeepAliveMessageFactory returns a factory function for creating keep-alive messages.
// It takes a MessageType and a function that generates the content of the message as parameters.
func NewKeepAliveMessageFactory(
	mt MessageType,
	contentFactory func() []byte,
) KeepAliveMessageFactory {
	return func() Message {
		return NewMessage(mt, contentFactory())
	}
}
