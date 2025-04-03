package libws

import (
	"net/url"

	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrConnectionClosed = errors.New("connection has been closed")
	ErrCannotConnect    = errors.New("connection cannot be established")
	ErrTerminated       = errors.New("program exit")
	ErrRateLimit        = errors.New("rate limit exceeded")
)

type ErrUnrecoverableConnection struct {
	err error
	url url.URL
}

func (e ErrUnrecoverableConnection) Error() string {
	return fmt.Sprintf("Unrecoverable connection error: %s to %s", e.err, e.url.String())
}

func (e ErrUnrecoverableConnection) Unwrap() error { return e.err }

func WrapErrorUnrecoverableConnection(err error, url url.URL) *ErrUnrecoverableConnection {
	if err != nil {
		return nil
	}
	return &ErrUnrecoverableConnection{
		err: err,
		url: url,
	}
}
