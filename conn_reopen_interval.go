package libws

import (
	"context"
	"sync"
	"time"
)

type (

	// reopenIntervalConnectionHandler is a ConnectionHandler implementation that
	// automatically reopens the connection after a fixed interval.
	// The struct uses a RWMutex to ensure safe concurrent access to the inner ConnectionHandler.
	reopenIntervalConnectionHandler struct {
		client Client

		connHandlerFactory ConnectionHandlerFactory

		reopenIntervalTicker *time.Ticker

		inner   ConnectionHandler
		innerMu sync.RWMutex

		logger logger

		handler MessageHandler

		closeC    CloseChan
		closeOnce sync.Once

		emitter emitter[EventType, EventType]
	}
)

// newReopenIntervalConn returns a new instance of reopenIntervalConnectionHandler.
// It takes a logger, the interval after which the connection should be reopened,
// and a ConnectionHandlerFactory as parameters.
func newReopenIntervalConn(
	logger logger,
	client Client,
	reopenIntervalTicker *time.Ticker,
	handler MessageHandler,
	emitter emitter[EventType, EventType],
	connFactory ConnectionHandlerFactory,
) *reopenIntervalConnectionHandler {
	return &reopenIntervalConnectionHandler{
		logger:               logger.WithField("type", "reopenIntervalConnectionHandler"),
		client:               client,
		reopenIntervalTicker: reopenIntervalTicker,
		connHandlerFactory:   connFactory,
		closeC:               make(CloseChan),
		emitter:              emitter,
		handler:              handler,
	}
}

// NewReopenIntervalConnFactory returns a function (ConnectionHandlerFactory) that
// creates a new instance of reopenIntervalConnectionHandler when called.
// It takes a logger, the interval ticker after which the connection should be reopened,
// and a ConnectionHandlerFactory as parameters.
func NewReopenIntervalConnFactory(
	logger logger,
	reopenInterval time.Duration,
	connFactory ConnectionHandlerFactory,
) ConnectionHandlerFactory {
	return func(client Client, handler MessageHandler, emitter emitter[EventType, EventType]) ConnectionHandler {
		return newReopenIntervalConn(
			logger,
			client,
			time.NewTicker(reopenInterval),
			handler,
			emitter,
			connFactory,
		)
	}
}

// Connect opens the initial connection and starts the run goroutine.
func (b *reopenIntervalConnectionHandler) Connect(ctx context.Context) error {
	b.logger.Infof("spawning and opening #0 conn")
	b.innerMu.Lock()
	b.inner = b.newConnectionHandler(ctx)
	b.innerMu.Unlock()
	go b.run(ctx)
	return nil
}

// Send sends a message to the server over the current connection.
func (b *reopenIntervalConnectionHandler) Send(m Message) {
	b.innerMu.RLock()
	b.inner.Send(m)
	b.innerMu.RUnlock()
}

// Recv receives a message from the server over the current connection.
func (b *reopenIntervalConnectionHandler) Recv(m Message) {
	b.innerMu.RLock()
	b.inner.Recv(m)
	b.innerMu.RUnlock()
}

// Close terminates the current connection and stops the run goroutine.
func (b *reopenIntervalConnectionHandler) Close() {
	b.safeClose()
}

// CloseChan returns a channel that can be used to receive a signal when the connection is closed.
func (b *reopenIntervalConnectionHandler) CloseChan() CloseChan {
	return b.closeC
}

// CloseErr returns the error that caused the connection to close.
func (b *reopenIntervalConnectionHandler) CloseErr() error {
	return b.inner.CloseErr()
}

func (b *reopenIntervalConnectionHandler) safeClose() {
	b.closeOnce.Do(b.close)
}

func (b *reopenIntervalConnectionHandler) close() {
	close(b.closeC)
	b.innerMu.RLock()
	b.inner.Close()
	b.innerMu.RUnlock()
}

// newConnectionHandler creates a new ConnectionHandler and attempts to establish a connection.
// If the connection attempt fails, it will retry indefinitely.
func (b *reopenIntervalConnectionHandler) newConnectionHandler(
	ctx context.Context,
) ConnectionHandler {
	for {
		conn := b.connHandlerFactory(b.client, b.handler, b.emitter)

		if err := conn.Connect(ctx); err != nil {
			b.logger.Errorf("conn user data stream was closed due to %s", err)
			// cleanup resources
			conn.Close()
			continue
		}

		return conn
	}
}

// run is a goroutine that manages reopening of the connection at a fixed interval,
// or when the current connection closes unexpectedly.
func (b *reopenIntervalConnectionHandler) run(ctx context.Context) {
	defer b.reopenIntervalTicker.Stop()

	connCount := 0
	closeChan := b.inner.CloseChan()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.reopenIntervalTicker.C:
			connCount++
			// Time to spawn a new conn. When a new one is opened, close the previous one. Order matters
			// to prevent data loss (duplicated data is preferred above lack of it)
			b.logger.Infof("spawning and opening #%d conn due to reopen trigger", connCount)

			nextConnectionHandler := b.newConnectionHandler(ctx)
			nextCloseChan := nextConnectionHandler.CloseChan()
			b.innerMu.Lock()
			b.inner.Close()
			b.inner = nextConnectionHandler
			b.innerMu.Unlock()
			closeChan = nextCloseChan
		case <-closeChan:
			connCount++
			b.logger.Infof(
				"spawning and opening #%d conn due to previous conn closed",
				connCount,
			)
			// inner conn closed unexpectedly. Open a new one
			conn := b.newConnectionHandler(ctx)
			closeChan = conn.CloseChan()
			b.innerMu.Lock()
			b.inner = conn
			b.innerMu.Unlock()
		}
	}
}
