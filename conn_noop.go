package libws

import "context"

type (
	emitter[K comparable, V any] interface {
		Emit(K, V)
	}

	// ConnectionHandler defines the interactions with a connection.
	ConnectionHandler interface {
		// Recv is called when a message from the server is received.
		// It handles the inbound data flow from the server.
		Recv(m Message)

		// Send is called when a message needs to be sent to the server.
		// It handles the outbound data flow towards the server.
		Send(m Message)

		// Connect establishes a connection to the server.
		// It is a blocking function which only returns when the connection is no longer active.
		Connect(ctx context.Context) error

		// CloseChan returns a channel that will be closed when the connection is closed.
		// This can be used to monitor the connection's closing event.
		CloseChan() CloseChan

		// CloseErr returns an error that explains why the connection was closed.
		// If the connection closed normally, CloseErr should return nil.
		CloseErr() error

		// Close closes the connection.
		// It should ensure that all resources related to the connection are cleaned up.
		Close()
	}

	// ConnectionHandlerFactory is a function type that takes a MessageHandler and an EventEmitter and returns a ConnectionHandler.
	ConnectionHandlerFactory func(Client, MessageHandler, emitter[EventType, EventType]) ConnectionHandler
)
