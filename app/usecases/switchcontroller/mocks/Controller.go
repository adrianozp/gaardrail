package mocks

import mock "github.com/stretchr/testify/mock"

type Controller struct {
	mock.Mock
}

func (_m *Controller) SetType(t string) error {
	ret := _m.Called(t)

	if len(ret) == 0 {
		panic("no return value specified for SetType")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(t)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Controller) Type() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Type")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

func NewController(t interface {
	mock.TestingT
	Cleanup(func())
}) *Controller {
	mock := &Controller{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
