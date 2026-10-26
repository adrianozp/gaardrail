package mocks

import (
	entities "github.com/adrianozp/gaardrail/app/entities"

	mock "github.com/stretchr/testify/mock"
)

type CreateMessageUseCase struct {
	mock.Mock
}

func (_m *CreateMessageUseCase) Create(_a0 entities.Message) (string, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(entities.Message) (string, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(entities.Message) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(entities.Message) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func NewCreateMessageUseCase(t interface {
	mock.TestingT
	Cleanup(func())
}) *CreateMessageUseCase {
	mock := &CreateMessageUseCase{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
