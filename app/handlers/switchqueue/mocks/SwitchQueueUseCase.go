package mocks

import mock "github.com/stretchr/testify/mock"

type SwitchQueueUseCase struct {
	mock.Mock
}

func (_m *SwitchQueueUseCase) Available() []string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Available")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

func (_m *SwitchQueueUseCase) Current() string {
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

func (_m *SwitchQueueUseCase) Switch(t string) error {
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

func NewSwitchQueueUseCase(t interface {
	mock.TestingT
	Cleanup(func())
}) *SwitchQueueUseCase {
	mock := &SwitchQueueUseCase{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
