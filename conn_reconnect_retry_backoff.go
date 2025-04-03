package libws

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type backoffCalculator func(attempts int) (time time.Duration)

type backoffConnectionHandler struct {
	client                Client
	emitter               emitter[EventType, EventType]
	logger                logger
	inner                 ConnectionHandler
	connHandlerFactory    ConnectionHandlerFactory
	calculator            backoffCalculator
	closeC                CloseChan
	closeOnce             sync.Once
	closeReason           error
	send                  chan Message
	recv                  chan Message
	handler               MessageHandler
	connDurationThreshold time.Duration
}

func (b *backoffConnectionHandler) newConnHandler(ctx context.Context) ConnectionHandler {
	var (
		attempts = 0
		ch       ConnectionHandler
	)

	for {
		attempts++

		ch = b.connHandlerFactory(b.client, b.handler, b.emitter)

		if err := ch.Connect(ctx); err != nil {
			if errors.Is(err, ErrCannotConnect) {
				b.logger.Infof("cannot connect, reconnecting asap due to: %s", err)
				// Try to establish the connection asap
				time.Sleep(time.Second)
				continue
			}

			ttw := b.calculator(attempts)
			b.logger.Infof("cannot connect after %s, waiting %s", err, ttw)
			time.Sleep(ttw)
			continue
		}

		return ch
	}
}

func (b *backoffConnectionHandler) run(ctx context.Context) {
	var (
		innerCloseChan = b.inner.CloseChan()
		attempts       = 0
		then           = time.Now().UTC()
	)

	defer b.inner.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.closeC:
			return
		case msg := <-b.recv:
			if b.inner != nil {
				// TODO: queue to buffer messages to send while reconnecting. Procrastinated as of now since
				// we are not sending messages to exchanges but ping/pongs
				b.inner.Recv(msg)
			}
		case msg := <-b.send:
			if b.inner != nil {
				// TODO: queue to buffer messages to send while reconnecting. Procrastinated as of now since
				// we are not sending messages to exchanges but ping/pongs
				b.inner.Send(msg)
			}
		case <-innerCloseChan:
			// Ensure resource clean-up
			b.inner.Close()
			b.closeReason = b.inner.CloseErr()

			if b.closeReason != nil {
				if errors.Is(b.closeReason, ErrConnectionClosed) ||
					errors.Is(b.closeReason, ErrTerminated) {
					// If this connection terminated because the connection died naturally, or we
					// have terminated it, reset counter to 0.
					delta := time.Since(then)
					if delta > b.connDurationThreshold {
						// We assume that the connection was healthy for `connDurationThreshold` and that it
						// was terminated due to natural reasons, so we should try to reconnect asap
						attempts = 0
					} else {
						attempts++
					}
				}
			}

			ttw := b.calculator(attempts)
			b.logger.Infof("retrying to connect after %s due to %s", ttw, b.closeReason)
			time.Sleep(ttw)

			// Reopen the client
			b.inner = b.newConnHandler(ctx)
			innerCloseChan = b.inner.CloseChan()
			then = time.Now().UTC()

			go b.emitter.Emit(EventReconnect, EventReconnect)
		}
	}
}

func (b *backoffConnectionHandler) Connect(ctx context.Context) error {
	// open the first connection synchronously.
	b.inner = b.newConnHandler(ctx)

	// once the first connection has been established, spawn goro and return.
	go b.run(ctx)

	return nil
}

func (b *backoffConnectionHandler) Recv(m Message) {
	b.recv <- m
}

func (b *backoffConnectionHandler) Send(m Message) {
	b.send <- m
}

func (b *backoffConnectionHandler) Close() {
	b.closeOnce.Do(func() {
		close(b.closeC)

		b.inner.Close()
	})
}

func (b *backoffConnectionHandler) CloseChan() CloseChan {
	return b.closeC
}

func (b *backoffConnectionHandler) CloseErr() error {
	return b.closeReason
}

func newBackoffConnectionHandler(
	logger logger,
	client Client,
	emitter emitter[EventType, EventType],
	connHandlerFactory ConnectionHandlerFactory,
	handler MessageHandler,
	calculator backoffCalculator,
	connDurationThreshold time.Duration,
) ConnectionHandler {
	return &backoffConnectionHandler{
		logger: logger.WithField(
			"type", "conn_handler_reconnect_exp_backoff",
		),
		client:                client,
		emitter:               emitter,
		handler:               handler,
		connHandlerFactory:    connHandlerFactory,
		calculator:            calculator,
		connDurationThreshold: connDurationThreshold,
		send:                  make(chan Message, 32),
		recv:                  make(chan Message, 32),
		closeC:                make(CloseChan),
	}
}

func NewBackoffConnectionHandlerFactory(
	logger logger,
	connHandlerFactory ConnectionHandlerFactory,
	calculator backoffCalculator,
	connDurationThreshold time.Duration,
) ConnectionHandlerFactory {
	return func(client Client, handler MessageHandler, emitter emitter[EventType, EventType]) ConnectionHandler {
		return newBackoffConnectionHandler(
			logger,
			client,
			emitter,
			connHandlerFactory,
			handler,
			calculator,
			connDurationThreshold,
		)
	}
}

func ExponentialBackoff(attempts int) float64 {
	return (math.Pow(2.0, float64(attempts)) - 1) / 2
}

func ExponentialBackoffSeconds(attempts int) time.Duration {
	return time.Duration(ExponentialBackoff(attempts)) * time.Second
}
