package mocks

import (
	context "context"

	entities "github.com/adrianozp/gaardrail/app/entities"
	mock "github.com/stretchr/testify/mock"
)

type Target struct {
	mock.Mock
}

func (_m *Target) Push(_a0 context.Context, _a1 entities.Message) (entities.Response, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Push")
	}

	var r0 entities.Response
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, entities.Message) (entities.Response, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, entities.Message) entities.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(entities.Response)
	}

	if rf, ok := ret.Get(1).(func(context.Context, entities.Message) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func NewTarget(t interface {
	mock.TestingT
	Cleanup(func())
}) *Target {
	mock := &Target{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
