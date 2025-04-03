package libws

import (
	"context"
)

type (
	// Client is the interface that defines the behavior of a client. This includes opening and closing connections,
	// sending messages, and managing event notifications.
	Client interface {
		// Open establishes a connection with the server
		Open(ctx context.Context) error
		// Send sends a message to the server
		Send(m Message)
		// Close closes the connection with the server
		Close()
		// CloseChan returns a channel that signals when the connection is closed
		CloseChan() CloseChan
	}

	CloseChan chan struct{}

	MessageHandler func(Client, Message)

	EventHandler func(Client, EventType)

	ClientFactory func() Client
)
