package libws

import (
	"sync"
	"time"

	"context"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/fasthttp/websocket"
)

type (
	openConnectionParamsRepo interface {
		Get(ctx context.Context) (OpenConnectionParams, error)
	}

	OpenConnectionParams struct {
		URL    url.URL
		Header http.Header
	}

	ErrAdapter func(*websocket.Conn, *http.Response, error) error

	ErrorAdapters struct {
		OnDial ErrAdapter
	}

	// WsConnection represents a WebSocket connection.
	// It implements the Connection interface.
	WsConnection struct {
		errAdapters              ErrorAdapters
		openConnectionParamsRepo openConnectionParamsRepo
		logger                   logger
		dialer                   *websocket.Dialer
		conn                     *websocket.Conn
		closeChan                CloseChan
		closeOnce                sync.Once
		closeReason              error
		closeReasonOnce          sync.Once
		recv                     chan<- Message // recv messages to be received over the wire
		send                     chan Message   // send messages to be sent over the wire
	}
)

var (
	NoopOpenConnectionParams = OpenConnectionParams{}
)

func NewWebsocketConnection(
	dialer *websocket.Dialer,
	openParamsRepo OpenConnectionParamsRepo,
	logger logger,
	recvChan chan<- Message,
	errorHandlers ErrorAdapters,
) *WsConnection {
	return &WsConnection{
		errAdapters:              errorHandlers,
		dialer:                   dialer,
		openConnectionParamsRepo: openParamsRepo,
		recv:                     recvChan,
		send:                     make(chan Message),
		closeChan:                make(CloseChan),
		logger:                   logger.WithField("net", "ws_connection"),
	}
}

func NewWebsocketFactory(
	logger logger,
	dialer *websocket.Dialer,
	openConnectionParamsRepo OpenConnectionParamsRepo,
	errorHandlers ErrorAdapters,
) ConnectionFactory {
	return func(ctx context.Context, recvChan chan<- Message) Connection {
		return NewWebsocketConnection(
			dialer,
			openConnectionParamsRepo,
			logger,
			recvChan,
			errorHandlers,
		)
	}
}

// Write sends a message over the WebSocket connection.
func (w *WsConnection) Write(m Message) error {
	w.send <- m
	return nil
}

// Close terminates the WebSocket connection.
// It ensures that all resources related to the connection are cleaned up.
func (w *WsConnection) Close() {
	w.safeClose()
}

// Open initiates the WebSocket connection.
// This method is blocking and returns when the connection is successfully established or an error occurs.
func (w *WsConnection) Open(ctx context.Context) error {
	return w.start(ctx)
}

// CloseChan returns a channel that will be closed when the WebSocket connection is closed.
// This can be used to monitor the connection's closing event.
func (w *WsConnection) CloseChan() CloseChan {
	return w.closeChan
}

// CloseErr returns an error that explains why the WebSocket connection was closed.
// If the connection closed normally, CloseErr should return nil.
func (w *WsConnection) CloseErr() error {
	return w.closeReason
}

func (w *WsConnection) start(ctx context.Context) error {
	p, err := w.openConnectionParamsRepo.Get(ctx)

	if err != nil {
		w.logger.Errorf("cannot get connection params due to %s: ", err)
		return err
	}

	conn, resp, err := w.dialer.Dial(p.URL.String(), p.Header)

	if err = w.handleDialError(conn, resp, err); err != nil {
		w.logger.Errorf("connection err to %s: %s, %+v", p.URL.String(), err, resp)
		return err
	}

	w.logger.Debugf("success opening connection to %s", p.URL.String())

	w.conn = conn

	// Override control message handlers to gain full control over 'control' frames, as
	// some exchange rate-limit its reception as well.
	conn.SetPingHandler(func(appData string) error {
		w.logger.Debugln("<= [PING]")
		w.recv <- NewPingMessage([]byte(appData))
		return nil
	})

	conn.SetPongHandler(func(appData string) error {
		w.logger.Debugln("<= [PONG]")
		w.recv <- NewPongMessage([]byte(appData))
		return nil
	})

	conn.SetCloseHandler(func(code int, text string) error {
		w.logger.Debugln("<= [CLOSE]")
		w.recv <- NewCloseMessage(code, []byte(text))
		return nil
	})

	go w.read(ctx)
	go w.write(ctx)

	return nil
}

func (w *WsConnection) read(ctx context.Context) {
	defer w.safeClose()

	for {
		select {
		case <-w.closeChan:
			w.setCloseReason(ErrTerminated)
			return
		case <-ctx.Done():
			w.setCloseReason(ErrTerminated)
			return
		default:
			messageType, bts, err := w.conn.ReadMessage()
			if err != nil {
				w.logger.Errorf("error occurred on websocket read: %s", err)

				w.setCloseReason(errors.Wrap(
					ErrConnectionClosed,
					"error occurred on websocket read: "+err.Error(),
				))
				return
			}
			// message types from ReadMessage are either binary or text
			switch messageType {
			case websocket.BinaryMessage:
				w.logger.Debugln("<= [BIN]")
				w.recv <- NewBinaryMessage(bts)
			case websocket.CloseMessage:
				w.logger.Debugln("<= [CLOSE]")
				w.recv <- NewCloseMessage(messageType, bts)
			default:
				w.logger.Debugf("<= [DATA] %s", string(bts))
				w.recv <- NewDataMessage(bts)
			}
		}
	}
}

func (w *WsConnection) write(ctx context.Context) {
	defer w.safeClose()

	for {
		select {
		case <-w.closeChan:
			w.setCloseReason(ErrTerminated)
			return
		case <-ctx.Done():
			w.setCloseReason(ErrTerminated)
			return
		case msg, ok := <-w.send:
			if !ok {
				w.logger.Infoln("closing connection from our side")
				_ = w.conn.WriteMessage(websocket.CloseMessage, []byte{})
				w.setCloseReason(ErrTerminated)
				return
			}

			deadline := time.Now().Add(time.Second)
			_ = w.conn.SetWriteDeadline(deadline)

			var err error

			switch msg.Type() {
			case PingMessage:
				w.logger.Debugln("=> [PING]")
				err = w.conn.WriteControl(websocket.PingMessage, msg.Data(), deadline)
				if e, ok := err.(net.Error); ok && e.Temporary() {
					err = nil
				}
			case PongMessage:
				w.logger.Debugln("=> [PONG]")
				err = w.conn.WriteControl(websocket.PongMessage, msg.Data(), deadline)
			case DataMessage:
				w.logger.Infof("=> [DATA] %s", msg.Data())
				err = w.conn.WriteMessage(websocket.TextMessage, msg.Data())
			}

			if err != nil {
				if websocket.IsCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure,
				) {
					w.setCloseReason(ErrConnectionClosed)
				} else {
					w.setCloseReason(errors.Wrap(ErrConnectionClosed, err.Error()))
				}
			}
		}
	}
}

func (w *WsConnection) safeClose() {
	w.closeOnce.Do(w.close)
}

func (w *WsConnection) close() {
	_ = w.conn.Close()
	close(w.closeChan)
}

func (w *WsConnection) setCloseReason(err error) {
	w.closeReasonOnce.Do(func() {
		w.closeReason = err
	})
}

func (w *WsConnection) handleDialError(conn *websocket.Conn, resp *http.Response, err error) error {
	if w.errAdapters.OnDial != nil {
		return w.errAdapters.OnDial(conn, resp, err)
	}

	// 1. Check HTTP errors first
	var msg string

	if resp != nil {
		if resp.Body != nil {
			bts, err := io.ReadAll(resp.Body)
			if err == nil {
				msg = string(bts)
			}
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return errors.Wrap(ErrRateLimit, msg)
		}
	}

	// 2. Network errors
	if err != nil {
		return errors.Wrap(ErrCannotConnect, err.Error())
	}

	return nil
}
