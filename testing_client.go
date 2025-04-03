package libws

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type mockClient struct {
	mock.Mock

	tapOpen func()
}

func (m *mockClient) Open(ctx context.Context) error {
	if m.tapOpen != nil {
		m.tapOpen()
	}
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockClient) Send(msg Message) {
	m.Called(msg)
}

func (m *mockClient) Close() {
	m.Called()
}

func (m *mockClient) CloseChan() CloseChan {
	args := m.Called()
	return args.Get(0).(CloseChan)
}
