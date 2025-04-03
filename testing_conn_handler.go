package libws

import "context"

type mockConnectionHandler struct {
	ConnectFunc   func(ctx context.Context) error
	CloseFunc     func()
	SendFunc      func(m Message)
	RecvFunc      func(m Message)
	CloseChanFunc func() CloseChan
	CloseErrFunc  func() error
}

func (m *mockConnectionHandler) Connect(ctx context.Context) error {
	return m.ConnectFunc(ctx)
}

func (m *mockConnectionHandler) Close() {
	m.CloseFunc()
}

func (m *mockConnectionHandler) Send(msg Message) {
	m.SendFunc(msg)
}

func (m *mockConnectionHandler) Recv(msg Message) {
	m.RecvFunc(msg)
}

func (m *mockConnectionHandler) CloseChan() CloseChan {
	return m.CloseChanFunc()
}

func (m *mockConnectionHandler) CloseErr() error {
	return m.CloseErrFunc()
}

type mockMessageHandler struct {
	HandleMessageFunc func(m Message)
}

func (m *mockMessageHandler) HandleMessage(msg Message) {
	m.HandleMessageFunc(msg)
}
