package libws

import (
	"context"
)

type noopConnection struct{}

func (w *noopConnection) Write(m Message) error {
	return nil
}

func (w *noopConnection) Close() {}

func (w *noopConnection) CloseChan() CloseChan { return nil }

func (w *noopConnection) CloseErr() error { return nil }

func (w *noopConnection) Open(context.Context) error { return nil }
