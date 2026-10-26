package mocks

import mock "github.com/stretchr/testify/mock"

type SwitchControllerUseCase struct {
	mock.Mock
}

func (_m *SwitchControllerUseCase) Current() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Current")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

func (_m *SwitchControllerUseCase) Switch(t string) error {
	ret := _m.Called(t)

	if len(ret) == 0 {
		panic("no return value specified for Switch")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(t)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func NewSwitchControllerUseCase(t interface {
	mock.TestingT
	Cleanup(func())
}) *SwitchControllerUseCase {
	mock := &SwitchControllerUseCase{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
