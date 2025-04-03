package libws

import (
	"context"
)

type (
	OpenConnectionParamsGetter func(ctx context.Context) (OpenConnectionParams, error)

	OpenConnectionParamsRepo struct {
		logger logger
		getter OpenConnectionParamsGetter
	}
)

func (r OpenConnectionParamsRepo) Get(
	ctx context.Context,
) (params OpenConnectionParams, err error) {
	params, err = r.getter(ctx)
	if err != nil {
		r.logger.Errorf("cannot fetch open connection params: %s", err)
	}
	return
}

func NewOpenConnectionParamsRepo(
	logger logger,
	getter OpenConnectionParamsGetter,
) OpenConnectionParamsRepo {
	return OpenConnectionParamsRepo{getter: getter, logger: logger}
}
