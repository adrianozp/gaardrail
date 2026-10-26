package mocks

import (
	context "context"

	entities "github.com/adrianozp/gaardrail/app/entities"
	mock "github.com/stretchr/testify/mock"
)

type MetricsReader struct {
	mock.Mock
}

func (_m *MetricsReader) Read(ctx context.Context) (entities.Metrics, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Read")
	}

	var r0 entities.Metrics
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (entities.Metrics, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) entities.Metrics); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(entities.Metrics)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func NewMetricsReader(t interface {
	mock.TestingT
	Cleanup(func())
}) *MetricsReader {
	mock := &MetricsReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
